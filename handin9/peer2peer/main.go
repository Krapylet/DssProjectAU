package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"./lottery"
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

var transactionsSinceLastBlock []string

var transactionCounter int64 = 0

var blockTree = make(map[string]BlockStruct)

var blocksMissingTransactions []BlockStruct
var blocksMissingBlocks []BlockStruct

var currentBlock BlockStruct

// Mutexes
var addressesLock = new(sync.RWMutex)
var connsLock = new(sync.RWMutex)
var msgIDCounterLock = new(sync.RWMutex)
var myAddressLock = new(sync.RWMutex)
var MessagesSeenLock = new(sync.RWMutex)
var currentBlockLock = new(sync.RWMutex)
var transactionsReceivedLock = new(sync.RWMutex)
var applyTransactionsLock = new(sync.RWMutex)
var transactionCounterLock = new(sync.RWMutex)
var testLock = new(sync.RWMutex)
var blocksMissingBlocksLock = new(sync.RWMutex)
var blocksMissingTransactionsLock = new(sync.RWMutex)

var ledger *account.Ledger = account.MakeLedger()

var MessageIDCounter = 0

var myName string
var myPk RSA.PublicKey
var mySk RSA.SecretKey

var gBlock GenesisBlockStruct

// Use when getting a new connection
type ConnectStruct struct {
	PeersList []string
	PKMap     map[string]RSA.PublicKey
}

type NewConnectionStruct struct {
	Address string
	PK      RSA.PublicKey
}

// Block struct
type BlockStruct struct {
	ID               string
	PK               RSA.PublicKey // vk
	TransactionsList []string      // U
	Slot             int64         // slot
	PreviousBlockID  string        // h
	Draw             *big.Int      // draw
}

// Special genesis block which holds other values than normal blocks
type GenesisBlockStruct struct {
	ID          string
	SpecialKeys map[RSA.PublicKey]RSA.SecretKey
	Seed        int64
}

func makeGenesisBlock() GenesisBlockStruct {
	gBlock = *new(GenesisBlockStruct)
	gBlock.Seed = 3
	gBlock.ID = "genesis"
	gBlock.SpecialKeys = Read10KeysFromFile()
	return gBlock
}

// Returns a list from block N to the genesis block, inlcuding N and the genesis block
func PathToN(N BlockStruct) ([]BlockStruct, bool) {
	if N.ID == "genesis" {

		return []BlockStruct{N}, true
	}
	prevN, exists := blockTree[N.PreviousBlockID] 
	if exists { 
		path, reachGenesis := PathToN(prevN)
		return append(path, N), reachGenesis 
	}	else 	{
		return []BlockStruct{N}, false
	}
}

func BranchLength(N BlockStruct) (int, bool) {
	val, reachGenesis := PathToN(N)
	return len(val), reachGenesis
}

func ChangeBranchTo(N BlockStruct) {
	if N.ID == "genesis" {
		currentBlock = blockTree["genesis"]
		ledger.Reset()
		give1MillionAU()
	} else {
		ChangeBranchTo(blockTree[N.PreviousBlockID])
		applyBlockTransactions(N)
	}
}

// read 10 specieal RSA keys from "special_keys.txt" and set their account to 1.000.000 AU
func Read10KeysFromFile() map[RSA.PublicKey]RSA.SecretKey {

	//keys -> marshall -> string with ; and : -> write bytearray
	//read bytearray -> cast to string, and split on ; and : -> unmarshall -> keys

	data, err := ioutil.ReadFile("special_keys.txt")

	if err != nil {
		panic("Could not read special_keys.txt. Error message: " + err.Error())
	}

	dataString := string(data)
	dataArray := strings.Split(dataString, ";")

	var specialKeys = make(map[RSA.PublicKey]RSA.SecretKey)

	for i := 0; i < 10; i++ {
		mpk := dataArray[i*2]
		msk := dataArray[1+i*2]
		var pk RSA.PublicKey
		var sk RSA.SecretKey
		PKerr := json.Unmarshal([]byte(mpk), &pk)
		SKerr := json.Unmarshal([]byte(msk), &sk)

		if PKerr != nil {
			panic("Error reading the " + strconv.Itoa(i) + "'th PK in special_keys.txt. Error message: " + PKerr.Error())
		}
		if SKerr != nil {
			panic("Error reading the " + strconv.Itoa(i) + "'th SK in special_keys.txt. Error message: " + SKerr.Error())
		}

		specialKeys[pk] = sk
	}

	return specialKeys
}

func readBolcksFromFile() list.List {
	data, err := ioutil.ReadFile("fake_blocks.log")

	if err != nil {
		panic("Could not read fake_blocks.log. Error message: " + err.Error())
	}

	dataString := string(data)
	dataArray := strings.Split(dataString, ";")

	var blocks = list.New()

	var i = 0
	for n, mb := range dataArray {
		if n == len(dataArray)-1 {
			continue
		}

		var b BlockStruct
		Berr := json.Unmarshal([]byte(mb), &b)

		if Berr != nil {
			panic("Error reading the " + strconv.Itoa(i) + "'th block in fake_blocks.log. Error message: " + Berr.Error())
		}

		blocks.PushBack(b)
		i++
	}

	return *blocks
}

func applyFakeBlocks() {
	fmt.Println("Reading fake blocks...")
	var blocks = readBolcksFromFile()
	fmt.Println("Read " + strconv.Itoa(blocks.Len()) + " fake blocks")
	fmt.Println("Applying fake blocks")
	for b := blocks.Front(); b != nil; b = b.Next() {
		//applyBlockTransactions(b.Value.(BlockStruct))
		blocksMissingTransactionsLock.Lock()
		blocksMissingTransactions = append(blocksMissingTransactions, b.Value.(BlockStruct))
		SendMessageToAll("NEWBLOCK", b.Value.(BlockStruct))
		blocksMissingTransactionsLock.Unlock()
		applyQueue()
	}
}

func logBlock(b BlockStruct) {
	//Check if output file already exists
	f, err := os.OpenFile("fake_blocks.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic("Could not open or create fake_blocks.log. Error message: " + err.Error())
	}
	defer f.Close()

	var mb []byte
	mb, err = json.Marshal(b)

	if err != nil {
		panic("Could not marshall blog while logging. Error message: " + err.Error())
	}

	var text = string(mb) + ";"

	//Write text to file
	var _, err2 = f.WriteString(text)

	if err != nil {
		panic("Could not write to fake_blocks.log. Error message: " + err2.Error())
	}
}

func logTrans() {
	//Check if output file already exists
	f, err := os.OpenFile("transactions.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic("Could not open or create transactions.log. Error message: " + err.Error())
	}
	defer f.Close()

	var mb []byte
	mb, err = json.Marshal(transactionsReceived)

	if err != nil {
		panic("Could not marshall transactionsList while logging. Error message: " + err.Error())
	}

	//Write text to file
	var _, err2 = f.Write(mb)

	if err != nil {
		panic("Could not write to transactions.log. Error message: " + err2.Error())
	}
}

func applyTransLog() {
	data, err := ioutil.ReadFile("transactions.log")

	if err != nil {
		panic("Could not read transactions.log. Error message: " + err.Error())
	}

	transactionsReceivedLock.Lock()
	err = json.Unmarshal(data, &transactionsReceived)
	if err != nil {
		panic("Could not unmarshal transactions.log. Error message: " + err.Error())
	}
	transactionsReceivedLock.Unlock()

}

func give1MillionAU() {
	for pk := range gBlock.SpecialKeys {
		name := ledger.EncodePK(pk)
		ledger.Accounts[name] = 1000000
	}
}

func runLottery(pk RSA.PublicKey) {
	var slotCounter int64
	slotCounter = 0
	for {
		seed := gBlock.Seed
		myDraw := lottery.Draw(seed, slotCounter, gBlock.SpecialKeys[pk])
		if lottery.HasWonLottery(myDraw, pk, seed, slotCounter, 1000000) {
			sendBlock(pk, myDraw, slotCounter)
		}
		slotCounter++
		time.Sleep(time.Second)
	}
}

func main() {

	// generate RSA Key
	myPk, mySk = RSA.KeyGen(2048)

	gBlock = makeGenesisBlock()
	currentBlock = *new(BlockStruct)
	currentBlock.ID = "genesis"
	currentBlock.Slot = 0

	blockTree["genesis"] = currentBlock

	give1MillionAU()

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

		var splitMsg []string = strings.Split(msg, " ")

		//_________________DEBUG COMMANDS__________________
		if msg == "!A" {
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
		if msg == "!C" {
			fmt.Println("--- MY CONS ---")
			connsLock.RLock()
			for i := range conns {
				fmt.Println("-> " + conns[i].RemoteAddr().String())
			}
			connsLock.RUnlock()
		}

		if msg == "!L" {
			fmt.Println("--- MY LEDGER ---")
			for key, value := range ledger.Accounts {
				fmt.Println(key + ": " + strconv.Itoa(value) + " AU")
			}
			fmt.Println("-----------------")
		}

		if msg == "!N" {
			fmt.Println("--- MY NAME ---")
			fmt.Println(myName)
			fmt.Println("-----------------")
		}

		if msg == "!P" {
			fmt.Println("--- KNOWN PKS ---")
			var pks = ledger.GetPks()
			for encodedKey := range pks {
				fmt.Println(encodedKey)
			}
			fmt.Println("-----------------")
		}

		if msg == "!TESTPOS" {
			posTest()
		}
		if msg == "!TESTNEG" {
			negTest()
		}

		//______________________SET EACH ACCOUNT TO 1000_________________________
		if msg == "!GIVE" {
			for name := range ledger.GetPks() {
				ledger.Accounts[name] = 1000
			}
			SendMessageToAll("GIVE", "")
		}

		//______________________Log all transaction seen to a file__________________
		if msg == "!LOG" {
			logTrans()
		}

		//______________________Read Transaction from file transactions.log__________________
		if msg == "!READLOG" {
			applyTransLog()
		}

		//___________________ APPLY A BUNCH OF FAKE BLOCKS TO TRIGGER ROLLBACK____________
		if msg == "!FAKE" {
			applyFakeBlocks()
		}

		//________________ STARTS A LOTTERY FOR EACH SPECIAL KEY_________________
		if msg == "!LOTTERY" {
			fmt.Println("--- STARTING LOTTERY ---")
			for pk, _ := range gBlock.SpecialKeys {
				go runLottery(pk)
			}
		}

		//_______________END CONNECTING PHASE AND START SENDING BLOCKS__________
		// should only be used on the initial host
		//isEndConnPhaseCommand := splitMsg[0] == "!START"
		//if isEndConnPhaseCommand {
		//	go sendBlock()
		//}

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
				fmt.Println("Invalid SEND command: Need 3 arguments (Amount, from, to), but recieved " + strconv.Itoa(len(splitMsg)-1) + "arguments")
				continue
			}

			var amount, err = strconv.Atoi(splitMsg[1])

			// check if its a valid amount
			if err != nil {
				fmt.Println("Invalid SEND command: Amount is not an integer")
				continue
			}

			if amount < 0 {
				fmt.Println("Invalid SEND command: amount most be positive")
				continue
			}

			var t *account.SignedTransaction = new(account.SignedTransaction)
			// set values from input
			transactionCounterLock.RLock()
			t.ID = splitMsg[2] + ":" + strconv.FormatInt(transactionCounter, 10)
			transactionCounterLock.RUnlock()

			t.Amount = amount
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

			// try apply transaction
			// ledger.SignedTransaction(t)

			atomic.AddInt64(&transactionCounter, 1)

			// get id of transaction, t.id
			transactionsReceivedLock.Lock()
			transactionsReceived[t.ID] = *t
			transactionsSinceLastBlock = append(transactionsSinceLastBlock, t.ID)
			transactionsReceivedLock.Unlock()

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

	} else {
		fmt.Println("Connection Established!")
		defer hostConn.Close()

		// Receive message from your host
		go receiveMessage(hostConn)

		//Before the client has been assigned a proper port, says it has port NEW. Messages from NEW ports are never put into MessagesSeen
		myAddress = "127.0.0.1:NEW"

		// get addresses and PKMAP
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
		//println("msgrecieved: " + msgReceived)
		//println("len: " + strconv.Itoa(len(splitMsg)))
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
			transactionsSinceLastBlock = append(transactionsSinceLastBlock, t.ID)
			transactionsReceivedLock.Unlock()

			// Broadcast this transaction
			forward(msgReceived)

			// Check if this transactions was missing in any received blocks
			go applyQueue()

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
			for name := range ledger.GetPks() {
				ledger.Accounts[name] = 1000
			}
			forward(msgReceived)
			break
		case "NEWBLOCK":

			// Receives BlockStruct
			var newBlock BlockStruct
			err := json.Unmarshal(marshalledMsg, &newBlock)
			if err != nil {
				fmt.Println("Error unmarshalling at Received Msg: " + err.Error())
				break
			}

			// Check that the lottery is valid
			if lottery.VerifyDraw(newBlock.Draw, gBlock.Seed, newBlock.Slot, newBlock.PK) {
				// forward msg
				forward(msgReceived)

				blocksMissingTransactions = append(blocksMissingTransactions, newBlock)

				go applyQueue()
			}
			break
		}
	}
}

// Send a new BlockStruct
func sendBlock(pk RSA.PublicKey, draw *big.Int, slot int64) {
	// define a new block
	newBlock := new(BlockStruct)

	newBlock.PK = pk

	// Put all transactions seen since last block in this block
	transactionsReceivedLock.Lock()
	newBlock.TransactionsList = transactionsSinceLastBlock
	transactionsSinceLastBlock = make([]string, 0)
	transactionsReceivedLock.Unlock()

	newBlock.PreviousBlockID = currentBlock.ID
	newBlock.Draw = draw

	newBlock.Slot = slot

	// marshal to get bytes of block
	byteBlock, _ := json.Marshal(newBlock)

	hashedBlock := RSA.MakeSHA256Hex(byteBlock)

	// Set Block ID
	newBlock.ID = hashedBlock

	// Only use this function when you need to generate a new set of fake blocks
	//logBlock(*newBlock)

	SendMessageToAll("NEWBLOCK", newBlock)
	println("Len of transactionlist in block we are about to send: " + strconv.Itoa(len((*newBlock).TransactionsList)))
	// apply transactions locally
	go applyBlockTransactions(*newBlock)
}

// check if all transactions in block b has been seen
func checkBlockTransactions(b BlockStruct) bool {
	for i := 0; i < len(b.TransactionsList); i++ {
		if _, seen := transactionsReceived[b.TransactionsList[i]]; !seen {
			return false
		}
	}
	return true	
}

// tries to apply next block in queue
func applyQueue() {
	blocksMissingTransactionsLock.Lock()
	defer blocksMissingTransactionsLock.Unlock()
	if (len(blocksMissingTransactions) == 0) {
		return
	}
	for i := 0; i < len(blocksMissingTransactions); i ++ {
		if checkBlockTransactions(blocksMissingTransactions[i]) {
			applyBlockTransactions(blocksMissingTransactions[i])
			// remove block i
			
			blocksMissingTransactions = append(blocksMissingTransactions[:i], blocksMissingTransactions[i+1:]...)
		}
	}

	blocksMissingBlocksLock.Lock()
	defer blocksMissingBlocksLock.Unlock()
	for i := 0; i < len(blocksMissingBlocks); i++ {
		_, reachedGenesis := PathToN(blocksMissingBlocks[i])
		if reachedGenesis {
			applyBlockTransactions(blocksMissingBlocks[i])
			if len(blocksMissingBlocks) - 1 >= i {
				blocksMissingBlocks = append(blocksMissingBlocks[:i], blocksMissingBlocks[i+1:]...)			
			} else {
				blocksMissingBlocks = blocksMissingBlocks[:i]
			}
			//If we find a block that can be applied, we need to look through all the previous blocksmissingblocks again
			i = -1
		}
	}
}


// apply transaction given by the block locally
func applyBlockTransactions(newBlock BlockStruct) {
	applyTransactionsLock.Lock()
	fmt.Println("Applying block| slot: " + strconv.FormatInt(newBlock.Slot, 10) + "; ID: " + newBlock.ID)

	// inserting block in branch
	blockTree[newBlock.ID] = newBlock



	// reset since last block
	transactionsReceivedLock.Lock()
	transactionsSinceLastBlock = make([]string, 0)
	transactionsReceivedLock.Unlock()

	// check if new block can be applyed to the current branch
	currentBlockLock.RLock()
	var isBranching = newBlock.PreviousBlockID != currentBlock.ID
	var currentLen, _ = BranchLength(currentBlock)
	currentBlockLock.RUnlock()

	if isBranching {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>New block doesn't point to current block. Checking if new block branch is longer")

		//Check wether the other block creates a longer branch, and switch if it does.

		var alternativeLen, reachGenesis  = BranchLength(newBlock)
		var switchBranch = currentLen < alternativeLen

		if !reachGenesis {
			blocksMissingBlocksLock.Lock()
			blocksMissingBlocks = append(blocksMissingBlocks, newBlock)
			blocksMissingBlocksLock.Unlock()
			applyTransactionsLock.Unlock()
			return
		}

		applyTransactionsLock.Unlock()
		if switchBranch {
			fmt.Println("New block has longer branch: Rolling back and applying new block branch")
			ChangeBranchTo(newBlock)
		}

		return
	}
	
	currentBlockLock.Lock()
	currentBlock = newBlock
	currentBlockLock.Unlock()

	transactionsList := currentBlock.TransactionsList

	println("Is about to apply " + strconv.Itoa(len(transactionsList)) + " transactions")

	transactionsReceivedLock.Lock()
	// list of: id-amount-from-to
	for _, t := range transactionsList {
		// get transaction, will panic if this transaction was not received
		if transaction, inMap := transactionsReceived[t]; inMap {
			ledger.SignedTransaction(&transaction)
		} else {
			panic("Didnt receive that transaction: " + t)
		}

	}

	//Apply bonus to whoever made the block
	ledger.GiveBonus(currentBlock.PK, len(transactionsList))

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
	senderName := ""

	for pk, _ := range gBlock.SpecialKeys {
		// skip your own account
		if senderName == "" {
			senderName = ledger.EncodePK(pk)
			continue
		}
		// create a signed transaction
		t := new(account.SignedTransaction)
		t.From = senderName
		t.ID = t.From + ":" + strconv.FormatInt(transactionCounter, 10)
		atomic.AddInt64(&transactionCounter, 1)
		t.Amount = 50
		t.To = ledger.EncodePK(pk)
		// encode transaction as a byte array
		toSign, _ := json.Marshal(t)
		// Create big int from this
		toSignBig := new(big.Int).SetBytes(toSign)
		// Sign using SK
		signature := RSA.Sign(*toSignBig, gBlock.SpecialKeys[pkMap[senderName]])
		// set signature
		t.Signature = signature.String()

		transactionsReceivedLock.Lock()
		transactionsReceived[t.ID] = *t
		transactionsSinceLastBlock = append(transactionsSinceLastBlock, t.ID)
		transactionsReceivedLock.Unlock()
		// Broadcast
		SendMessageToAll("TRANSACTION", t)
	}
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
