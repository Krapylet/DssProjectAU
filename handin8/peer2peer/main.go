package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"../RSA"
	"./account"
)

// String containing your ip and port "ip:port"
var myAddress string

// All active connections of this peer
var conns []net.Conn

// Addresses of all known peers
var addresses []string

// MessagesSeen Set of all messages sent
var MessagesSeen = make(map[string]bool)

// Channel to determine if the tcp listener is running
var tcpListenerRunning = make(chan bool)

// Channel for initial conn data
var gotConnData = make(chan bool)

// MAP of key=transaction (with no signature), value=signature. Transaction seen since last BlockStruct...
var transactionsReceived = make(map[string]account.SignedTransaction)

var blockCounter int64 = 0
var transactionCounter int64 = 0

// Mutexes
var addressesLock = new(sync.RWMutex)
var connsLock = new(sync.RWMutex)
var msgIDCounterLock = new(sync.RWMutex)
var myAddressLock = new(sync.RWMutex)
var MessagesSeenLock = new(sync.RWMutex)
var sequencerAddressLock = new(sync.RWMutex)
var blockCounterLock = new(sync.RWMutex)
var transactionsReceivedLock = new(sync.RWMutex)
var applyTransactionsLock = new(sync.RWMutex)
var transactionCounterLock = new(sync.RWMutex)
var testLock = new(sync.RWMutex)

var ledger *account.Ledger = account.MakeLedger()

var MessageIDCounter = 0

var myName string
var myPk RSA.PublicKey
var mySk RSA.SecretKey

var sequencerAddress string
var sequencerPK RSA.PublicKey

// only know be the original host
var sequencerSK RSA.SecretKey

// Use when getting a new connection
type ConnectStruct struct {
	PeersList   []string
	PKMap       map[string]RSA.PublicKey
	Sequencer   string
	SequencerPK RSA.PublicKey
}

type NewConnectionStruct struct {
	Address string
	PK      RSA.PublicKey
}

// Block struct
type BlockStruct struct {
	Number           int64
	TransactionsList []string
}

type BlockSenderStruct struct {
	Signature []byte
	Block     []byte
}

func main() {
	// generate RSA Key
	myPk, mySk = RSA.KeyGen(2048)

	// Tries to connect, otherwise start own connection at port 20000
	connect()

	// Listen for incoming TCP connections
	// Split, 0 = ip, 1 = port
	myAddressLock.RLock()
	splitMyAddr := strings.Split(myAddress, ":")
	myAddressLock.RUnlock()

	go tcpListener(splitMyAddr[0], splitMyAddr[1])

	<-tcpListenerRunning

	// make transactions
	for {
		fmt.Print("To make a new transaction, use: SEND 'amount' 'from' 'to' > ")

		// Prompt for user input and send to all known connections
		msg, _ := bufio.NewReader(os.Stdin).ReadString('\n')

		//Trim msg of leading and trailing whitespace
		msg = strings.TrimSpace(msg)

		//_________________DEBUG COMMANDS__________________
		if strings.Contains(msg, "!A") {
			fmt.Println("--- KNOWN LISTENERS ---")
			myAddressLock.RLock()
			fmt.Println("(I LISTEN ON: " + myAddress + ")")
			myAddressLock.RUnlock()
			addressesLock.RLock()
			for i := range addresses {
				fmt.Println("-> " + addresses[i])
			}
			addressesLock.RUnlock()
		}
		if strings.Contains(msg, "!C") {
			fmt.Println("--- MY CONS ---")
			connsLock.RLock()
			for i := range conns {
				fmt.Println("-> " + conns[i].RemoteAddr().String())
			}
			connsLock.RUnlock()
		}

		if strings.Contains(msg, "!L") {
			fmt.Println("--- MY LEDGER ---")
			for key, value := range ledger.Accounts {
				fmt.Println(key + ": " + strconv.Itoa(value))
			}
			fmt.Println("-----------------")
		}

		if strings.Contains(msg, "!N") {
			fmt.Println("--- MY NAME ---")
			fmt.Println(myName)
			fmt.Println("-----------------")
		}

		if strings.Contains(msg, "!P") {
			fmt.Println("--- KNOWN PKS ---")
			var pks = ledger.GetPks()
			for encodedKey := range pks {
				fmt.Println(encodedKey)
			}
			fmt.Println("-----------------")
		}

		if strings.Contains(msg, "!SEQ") {
			fmt.Println("--- Sequencer Address ---")
			fmt.Println(sequencerAddress)
			fmt.Println("-----------------")
		}

		if strings.Contains(msg, "!TESTPOS") {
			posTest()
		}
		if strings.Contains(msg, "!TESTNEG") {
			negTest()
		}

		var splitMsg []string = strings.Split(msg, " ")

		//______________________SET EACH ACCOUNT TO 1000_________________________
		if strings.Contains(msg, "!GIVE") {
			for name, _ := range ledger.GetPks() {
				ledger.Accounts[name] = 1000
			}
			SendMessageToAll("GIVE", "")
		}

		//_______________END CONNECTING PHASE AND START SENDING BLOCKS__________
		// should only be used on the initial host
		isEndConnPhaseCommand := splitMsg[0] == "!START"
		if isEndConnPhaseCommand {
			go sendBlock()
		}

		//_______________SEND X TRANSACTIONS TO B AND C__________________
		isSend1000Command := splitMsg[0] == "!SPAM"
		if isSend1000Command {
			amountOfTransactions, _ := strconv.Atoi(splitMsg[1])
			spamTest(amountOfTransactions)
		}

		//______________________TRANSACTION COMMAND___________________________
		// "SEND 'amount' 'from' 'to'"
		var isSendCommand bool = splitMsg[0] == "SEND"
		if isSendCommand {

			if len(splitMsg) != 4 {
				fmt.Println("Invalid SEND command")
				continue
			}

			var t *account.SignedTransaction = new(account.SignedTransaction)
			// set values from input
			myAddressLock.RLock()
			transactionCounterLock.RLock()
			t.ID = myAddress + ":" + strconv.FormatInt(transactionCounter, 10)
			myAddressLock.RUnlock()
			transactionCounterLock.RUnlock()

			t.Amount, _ = strconv.Atoi(splitMsg[1])
			t.From = splitMsg[2]
			t.To = strings.TrimSpace(splitMsg[3])

			// encode transaction as a byte array
			toSign, _ := json.Marshal(t)
			// Create big int from this
			toSignBig := new(big.Int).SetBytes(toSign)
			// Sign using SK
			signature := RSA.Sign(*toSignBig, mySk)

			// set signature
			t.Signature = signature.String()

			// check if its a valid amount
			if !(t.Amount > 0) {
				fmt.Println("Invalid Transaction, amount most be positive")
				continue
			}

			// try apply transaction
			// ledger.SignedTransaction(t)

			atomic.AddInt64(&transactionCounter, 1)

			// get id of transaction, t.id
			transactionNoSign := t.ID
			transactionsReceivedLock.Lock()
			transactionsReceived[transactionNoSign] = *t
			transactionsReceivedLock.Unlock()

			fmt.Println(t)

			// Broadcast
			SendMessageToAll("TRANSACTION", t)
		}

		//________________________QUIT COMMAND_______________________________
		// "QUIT"
		var isQuitCommand bool = splitMsg[0] == "QUIT"
		if isQuitCommand {
			println("Quitting")
			myAddressLock.RLock()
			SendMessageToAll("DISCONNECT", myAddress)
			myAddressLock.RUnlock()
			break
		}
	}
}

// Initial connect
func connect() {
	// Try to connect to existing Peer
	// Ask for IP and Port
	fmt.Println("Connect to IP...")
	remoteIP := "127.0.0.1"
	fmt.Println("Connect to port...")
	remotePort, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	remotePort = strings.TrimSpace(remotePort)

	fmt.Println("Trying to connect to: " + remoteIP + ":" + remotePort)
	hostConn, _ := net.Dial("tcp", remoteIP+":"+remotePort)

	if hostConn == nil {
		fmt.Println("No existing peer found at: " + remoteIP + ":" + remotePort)

		// Give ip and port of where to listen for TCP connections
		myIP := "127.0.0.1"
		myPort := "20000"

		// Set myAddress
		myAddress = myIP + ":" + myPort

		// add yourself to known peers on the network
		addresses = append(addresses, myIP+":"+myPort)

		// adds your pk to ledger pks list and returns your name
		myName = ledger.EncodePK(myPk)

		// Setup Sequencer
		sequencerAddress = myAddress
		fmt.Println("Generating Sequencer Key Pair")
		sequencerPK, sequencerSK = RSA.KeyGen(2048)

	} else {
		fmt.Println("Connection Established!")
		defer hostConn.Close()

		// Receive message from your host
		go receiveMessage(hostConn)

		//Before the client has been assigned a proper port, says it has port NEW. Messages from NEW ports are never put into MessagesSeen
		myAddress = "127.0.0.1:NEW"

		// get addresses, sequencer and PKMAP
		fmt.Println("Requesting Connection Struct data")
		SendMessage("GETCONNDATA", "", hostConn)
		// wait til received
		<-gotConnData

	}
}

// Listen for new tcp connections
func tcpListener(myIP string, myPort string) {
	// Open own port for incoming TCP
	// Local IP
	ln, err := net.Listen("tcp", ":"+myPort)
	if err != nil {
		panic(err.Error())
	}
	defer ln.Close()

	// TCP listener is running
	tcpListenerRunning <- true

	fmt.Println("LISTENING ON PORT -> " + myIP + ":" + myPort)
	for {
		conn, _ := ln.Accept()
		fmt.Println("Got a new connection from: " + conn.RemoteAddr().String())
		// New active connection
		connsLock.Lock()
		conns = append(conns, conn)
		connsLock.Unlock()
		// Setup message receiver for each new connection
		go receiveMessage(conn)
	}
}

// Sends a new message to known peers. This increases the messageIDCounter
func SendMessageToAll(typeString string, msg interface{}) {
	testLock.Lock()
	// Marshall the object that should be sent
	marshalledMsg, _ := json.Marshal(msg)
	// Calculate the message ID
	myAddressLock.RLock()
	msgIDCounterLock.RLock()
	var id string = myAddress + ":" + strconv.Itoa(MessageIDCounter)
	myAddressLock.RUnlock()
	msgIDCounterLock.RUnlock()

	msgIDCounterLock.Lock()
	MessageIDCounter++
	msgIDCounterLock.Unlock()

	msgIDCounterLock.RLock()
	combinedMsg := id + ";" + typeString + ";" + string(marshalledMsg) + "\n"
	msgIDCounterLock.RUnlock()
	// Insert message into map of known messages
	MessagesSeenLock.Lock()
	MessagesSeen[combinedMsg] = true
	MessagesSeenLock.Unlock()

	myAddressLock.RLock()
	//println("<< type: " + typeString + ", ID: " + myAddress)
	myAddressLock.RUnlock()
	// write msg to all known connections
	connsLock.RLock()
	for i := range conns {
		conns[i].Write([]byte(combinedMsg))
	}
	connsLock.RUnlock()
	testLock.Unlock()
}

// forwards messages received to known peers without chaning it
func forward(msg string) {
	// write msg to all known connections
	connsLock.RLock()
	for i := range conns {
		conns[i].Write([]byte(msg))
	}
	connsLock.RUnlock()
}

//Sends a message to a single peer. Only used for special purposes such as initialization. The same message can be sent multiple times
func SendMessage(typeString string, msg interface{}, conn net.Conn) {
	// Marshall the object that should be sent
	marshalledMsg, _ := json.Marshal(msg)

	myAddressLock.RLock()
	myAddr := myAddress
	myAddressLock.RUnlock()

	// Calculate message ID
	msgIDCounterLock.RLock()
	var id string = myAddr + ":" + strconv.Itoa(MessageIDCounter)
	msgIDCounterLock.RUnlock()
	// Prepend S to the message to show that it shouldn't be mapped
	combinedMsg := "S" + id + ";" + typeString + ";" + string(marshalledMsg) + "\n"
	//fmt.Println("<- type: " + typeString + ", to: " + conn.LocalAddr().String() + ", ID: " + myAddr)
	// write msg to target connection
	conn.Write([]byte(combinedMsg))
}

func receiveMessage(conn net.Conn) {
	// Keeps checking for new messages
	for {
		msgReceived, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			return
		}

		splitMsg := strings.Split(msgReceived, ";")
		typeString := splitMsg[1]
		ID := splitMsg[0]
		//fmt.Println("-> type: " + typeString + ", from: " + conn.LocalAddr().String() + ", ID: " + ID)
		//Break if we have already seen the message
		MessagesSeenLock.Lock()
		seen := MessagesSeen[msgReceived] && !strings.Contains(ID, "S")
		if seen {
			MessagesSeenLock.Unlock()
			continue
		}
		MessagesSeen[msgReceived] = true
		MessagesSeenLock.Unlock()

		// Messages have the format id;typeString;msg where msg can have any type
		marshalledMsg := []byte(strings.Join(splitMsg[2:], ";"))

		switch typeString {
		// Notification that a new peer has joined the network
		case "NEWCONNECTION":
			// msg is NewConnectionStruct
			var newConnStruct NewConnectionStruct
			err := json.Unmarshal(marshalledMsg, &newConnStruct)

			if err != nil {
				fmt.Println("Failed to unmarshal at NEWCONNECTION")
			}

			// check if the address is already known by this peer
			address := newConnStruct.Address
			addressesLock.Lock()
			if !strings.Contains(strings.Join(addresses, ","), address) {
				// Not known, so add to list of addresses
				addresses = append(addresses, address)
			}
			addressesLock.Unlock()

			// Add the new PK
			newPK := newConnStruct.PK
			ledger.EncodePK(newPK)

			// Send the message to all the known connections of this peer too
			forward(msgReceived)
			break
		// A transaction
		case "TRANSACTION":
			// Unmarshal the transaction
			var t account.SignedTransaction
			json.Unmarshal(marshalledMsg, &t)
			// Update ledger
			// ledger.SignedTransaction(&t)

			// add to seen
			transactionsReceivedLock.Lock()
			transactionsReceived[t.ID] = t
			transactionsReceivedLock.Unlock()

			// Broadcast this transaction
			forward(msgReceived)
			break
		// Receive a disconnect message
		case "DISCONNECT":
			// Remove connection from known connections
			connsLock.Lock()
			removeConn(conn)
			connsLock.Unlock()

			// Remove address from known addresses
			var address string
			json.Unmarshal(marshalledMsg, &address)

			addressesLock.Lock()
			removeAddress(address)
			addressesLock.Unlock()

			// Close connection
			conn.Close()
			break
		case "GETCONNDATA":
			fmt.Println("RECEIVED GETCONNDATA")
			connStruct := new(ConnectStruct)
			connStruct.PeersList = addresses
			connStruct.PKMap = ledger.GetPks()
			connStruct.Sequencer = sequencerAddress
			connStruct.SequencerPK = sequencerPK
			SendMessage("CONNDATA", connStruct, conn)
			break
		case "CONNDATA":
			fmt.Println("RECEIVED CONNDATA")
			var connData ConnectStruct
			err := json.Unmarshal(marshalledMsg, &connData)

			if err != nil {
				fmt.Println("Could not unmarshal at CONNDATA")
				continue
			}
			// set addresses
			addressesLock.Lock()
			addresses = connData.PeersList
			addressesLock.Unlock()
			// set known pks
			ledger.SetPks(connData.PKMap)
			// set sequencer address
			sequencerAddressLock.Lock()
			sequencerAddress = connData.Sequencer
			sequencerAddressLock.Unlock()
			// set sequencer pk
			sequencerPK = connData.SequencerPK

			// set my name
			myName = ledger.EncodePK(myPk)

			// append my own address
			myAddr := conn.LocalAddr().String()
			addresses = append(addresses, myAddr)

			// Disconnect from old host
			fmt.Println("Disconnection from host...")
			SendMessage("DISCONNECT", "", conn)

			connsLock.Lock()
			removeConn(conn)
			connsLock.Unlock()

			// connect to max 10 peers
			connectToPeers()

			// set my address
			myAddressLock.Lock()
			myAddress = myAddr
			myAddressLock.Unlock()

			// broadcast that you've connected
			// should contain -> address and encode(pk)
			newConnStruct := new(NewConnectionStruct)
			newConnStruct.Address = myAddr
			newConnStruct.PK = myPk
			SendMessageToAll("NEWCONNECTION", newConnStruct)

			gotConnData <- true
			break
		// Used to give each account 1000
		case "GIVE":
			for name, _ := range ledger.GetPks() {
				ledger.Accounts[name] = 1000
			}
			forward(msgReceived)
			break
		case "NEWBLOCK":
			// Received a new block of transactions from the sequencer

			// Receives BlockSenderStruct
			var received BlockSenderStruct
			err := json.Unmarshal(marshalledMsg, &received)
			if err != nil {
				fmt.Println("Error unmarshalling at Received Msg")
				break
			}
			// Byte arrays
			signature := received.Signature
			block := received.Block

			hashedBlock := RSA.MakeSHA256Hex(block)
			intBlock := new(big.Int).SetBytes([]byte(hashedBlock))
			intSignature := new(big.Int).SetBytes(signature)
			verified := RSA.Verify(*intSignature, *intBlock, sequencerPK)

			if !verified {
				fmt.Println("Wrong Signature on block")
				break
			}

			// unmarshal to block
			var newBlock BlockStruct
			err = json.Unmarshal(block, &newBlock)
			if err != nil {
				fmt.Println("Failed unmarshalling at NEWBLOCK")
				panic(err.Error())
			}

			// forward msg
			forward(msgReceived)

			// apply block
			go applyBlockTransactions(newBlock)

			break
		}
	}
}

// Send a new BlockStruct every 10 sec. Only the initial host runs this
func sendBlock() {
	// define a new block
	newBlock := new(BlockStruct)
	for {
		time.Sleep(time.Second * 10)
		blockCounterLock.RLock()
		newBlock.Number = blockCounter
		blockCounterLock.RUnlock()
		// create list of all transactions
		var listOfTransactions []string
		transactionsReceivedLock.RLock()
		for tid, _ := range transactionsReceived {
			listOfTransactions = append(listOfTransactions, tid)
		}
		transactionsReceivedLock.RUnlock()
		// Each transaction in the list is:
		// - id - Each peer are responsible for storing the signedTransaction
		newBlock.TransactionsList = listOfTransactions

		// Sign this block using sequencer SK
		// marshal to get bytes of block
		byteBlock, _ := json.Marshal(newBlock)
		// Create big.Int from this
		intBlock := new(big.Int).SetBytes(byteBlock)

		hashedIntBlock := []byte(RSA.MakeSHA256Hex(intBlock.Bytes()))

		// sign the intBlock
		toSign := new(big.Int).SetBytes(hashedIntBlock)
		signedBlock := RSA.Sign(*toSign, sequencerSK)

		// [0] = byte array of the signature
		// [1] = marshalled block
		toSend := new(BlockSenderStruct)
		toSend.Signature = signedBlock.Bytes()
		toSend.Block = byteBlock
		SendMessageToAll("NEWBLOCK", toSend)

		// apply transactions locally
		go applyBlockTransactions(*newBlock)
	}
}

// apply transaction given by the block locally
func applyBlockTransactions(block BlockStruct) {
	applyTransactionsLock.Lock()
	fmt.Println("Applying block", block.Number)
	// increment your block counter
	atomic.AddInt64(&blockCounter, 1)

	counter := block.Number
	transactionsList := block.TransactionsList

	// check if counters match: +1 since blockCounter is incremented at the beginning
	blockCounterLock.RLock()
	if blockCounter != (counter + 1) {
		fmt.Println("Wrong Block Counter")
		blockCounterLock.RUnlock()
		return
	}
	blockCounterLock.RUnlock()
	// list of: id-amount-from-to
	for _, t := range transactionsList {
		// get transaction, will panic if this transaction was not received
		if transaction, inMap := transactionsReceived[t]; inMap {
			ledger.SignedTransaction(&transaction)
		} else {
			panic("Didnt receive that transaction: " + t)
		}

	}
	// reset map of seen transactions
	transactionsReceivedLock.Lock()
	transactionsReceived = make(map[string]account.SignedTransaction)
	transactionsReceivedLock.Unlock()
	applyTransactionsLock.Unlock()
}

func connectToPeers() {
	// connect to, up to 10 newest connections, excluding your host
	connCounter := 0
	// Gets the last position in addresses which are not yourself
	addressesLock.RLock()
	pos := len(addresses) - 2

	for connCounter < 10 && pos >= 0 {
		// Makes sure we dont connect to our host again...
		// Try to connect to peer
		conn, _ := net.Dial("tcp", addresses[pos])

		if conn == nil {
			fmt.Println("Failed to connect to: " + addresses[pos])
		} else {
			fmt.Println("Connection established with: " + conn.RemoteAddr().String())
			// Add connection to active connections
			connsLock.Lock()
			conns = append(conns, conn)
			connsLock.Unlock()
			// Setup message receiver for this conn
			go receiveMessage(conn)
		}
		// Update connCounter, i.e. connected to a new peer.
		connCounter++
		// Get next position
		pos--
	}
	addressesLock.RUnlock()
}

// Removes a connection from the global array of connections
func removeConn(conn net.Conn) {
	// Create a temporary holder for valid connections
	var temp []net.Conn
	// Copy all connections except the one we want to remove into temp
	for i := range conns {
		if conns[i] != conn {
			temp = append(temp, conns[i])
		}
	}
	// Overwrite conns with temp
	conns = temp
}

// Removes an address from the global array of addresses
func removeAddress(address string) {
	// Create a temporary holder for valid connections
	var temp []string
	// Copy all connections except the one we want to remove into temp
	for i := range addresses {
		if addresses[i] != address {
			temp = append(temp, addresses[i])
		}
	}
	// Overwrite addresses with temp
	addresses = temp
}

// Automatic test for testing valid SignedTransactions
func posTest() {
	fmt.Println()
	fmt.Println("--- TESTING VALID SIGNED TRANSACTIONS ---")
	pkMap := ledger.GetPks()

	fmt.Println()
	fmt.Println("-- sending 50 to each account from my account --")
	// Send 50 to each known peer from your own account.
	for name, _ := range pkMap {
		// skip your own account
		if name == myName {
			continue
		}
		// create a signed transaction
		t := new(account.SignedTransaction)
		t.ID = myAddress + ":" + strconv.FormatInt(transactionCounter, 10)
		t.Amount = 50
		t.From = myName
		t.To = name
		// encode transaction as a byte array
		toSign, _ := json.Marshal(t)
		// Create big int from this
		toSignBig := new(big.Int).SetBytes(toSign)
		// Sign using SK
		signature := RSA.Sign(*toSignBig, mySk)
		// set signature
		t.Signature = signature.String()

		transactionsReceived[t.ID] = *t
		// Broadcast
		SendMessageToAll("TRANSACTION", t)
	}

	// Print the ledger values
	fmt.Println()
	fmt.Println("-- OUTPUT FROM TRANSACTIONS --")
	fmt.Println("--- MY LEDGER ---")
	for key, value := range ledger.Accounts {
		fmt.Println(key + ": " + strconv.Itoa(value))
	}
	fmt.Println("-----------------")
	fmt.Println()
}

// Try to send 100 from an account that is not yours
func negTest() {
	fmt.Println()
	fmt.Println("--- TESTING INVALID SIGNED TRANSACTIONS ---")

	pkMap := ledger.GetPks()

	fmt.Println()
	fmt.Println("-- sending 100 to my own account from each other account --")
	for name, _ := range pkMap {
		// skip your own account
		if name == myName {
			continue
		}
		// create a signed transaction
		t := new(account.SignedTransaction)
		t.Amount = 100
		t.From = name
		t.To = myName
		// encode transaction as a byte array
		toSign, _ := json.Marshal(t)
		// Create big int from this
		toSignBig := new(big.Int).SetBytes(toSign)
		// Sign using SK
		signature := RSA.Sign(*toSignBig, mySk)
		// set signature
		t.Signature = signature.String()
		// apply locally
		ledger.SignedTransaction(t)
		// Broadcast
		SendMessageToAll("TRANSACTION", t)
	}
	// Print the ledger values
	fmt.Println()
	fmt.Println("-- OUTPUT FROM TRANSACTIONS --")
	fmt.Println("--- MY LEDGER ---")
	for key, value := range ledger.Accounts {
		fmt.Println(key + ": " + strconv.Itoa(value))
	}
	fmt.Println("-----------------")
	fmt.Println()
}

func spamTest(number int) {
	fmt.Println("SpamTest")
	nameB := ""
	nameC := ""

	pkMap := ledger.GetPks()
	for name, _ := range pkMap {
		if name == myName {
			continue
		}
		if nameB == "" {
			nameB = name
		} else {
			nameC = name
		}
	}
	go makeXTransactions(nameB, number)
	go makeXTransactions(nameC, number)
}

func makeXTransactions(name string, number int) {
	fmt.Println("Sends", number, "of transactions")
	for i := 0; i < number; i++ {
		// create a signed transaction
		t := new(account.SignedTransaction)
		transactionCounterLock.Lock()
		t.ID = myAddress + ":" + strconv.FormatInt(transactionCounter, 10)
		atomic.AddInt64(&transactionCounter, 1)
		transactionCounterLock.Unlock()
		t.Amount = 1
		t.From = myName
		t.To = name
		// encode transaction as a byte array
		toSign, _ := json.Marshal(t)
		// Create big int from this
		toSignBig := new(big.Int).SetBytes(toSign)
		// Sign using SK
		signature := RSA.Sign(*toSignBig, mySk)
		// set signature
		t.Signature = signature.String()
		//fmt.Println(t)
		transactionsReceivedLock.Lock()
		transactionsReceived[t.ID] = *t
		transactionsReceivedLock.Unlock()
		// Broadcast
		SendMessageToAll("TRANSACTION", t)
	}
}
