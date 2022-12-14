package main

import (
	"../RSA"
	"./account"
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// String containing your ip and port "ip:port"
var myAddress string

// All active connections of this peer
var conns []net.Conn

// Addresses of all known peers
var addresses []string

// MessagesSeen Set of all messages sent
var MessagesSeen = make(map[string]bool)

// Bool to determine if the tcp listener is running
var tcpListenerRunning bool

// Flag for when list of connections is received
var gotConnsList bool

// Flag for when PKMap is received
var gotPKmap bool

// Mutexes
var addressesLock = new(sync.RWMutex)
var connsLock = new(sync.RWMutex)
var msgIDCounterLock = new(sync.RWMutex)
var myAddressLock = new(sync.RWMutex)
var MessagesSeenLock = new(sync.RWMutex)
var listenerLock = new(sync.RWMutex)
var gotConnsListLock = new(sync.RWMutex)
var gotPkMapLock = new(sync.RWMutex)

// Conditions for locks
var listenerCond = sync.NewCond(listenerLock.RLocker())
var gotConnsListCond = sync.NewCond(gotConnsListLock.RLocker())
var pkMapCond = sync.NewCond(gotPkMapLock.RLocker())

var ledger *account.Ledger = account.MakeLedger()

var MessageIDCounter = 0

var myName string
var myPk RSA.PublicKey
var mySk RSA.SecretKey

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

	listenerLock.RLock()
	for !tcpListenerRunning {
		listenerCond.Wait()
		listenerLock.RUnlock()
	}

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
				fmt.Println("-" + addresses[i])
			}
			addressesLock.RUnlock()
		}
		if strings.Contains(msg, "!C") {
			fmt.Println("--- MY CONS ---")
			connsLock.RLock()
			for i := range conns {
				fmt.Println("-" + conns[i].RemoteAddr().String())
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

		if strings.Contains(msg, "!TESTPOS") {
			posTest()
		}
		if strings.Contains(msg, "!TESTNEG") {
			negTest()
		}

		var splitMsg []string = strings.Split(msg, " ")

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
			msgIDCounterLock.RLock()
			t.ID = myAddress + ":" + strconv.Itoa(MessageIDCounter)
			myAddressLock.RUnlock()
			msgIDCounterLock.RUnlock()

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
			ledger.SignedTransaction(t)

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
		//myAddressLock.Lock()
		myAddress = "127.0.0.1:NEW"
		//myAddressLock.Unlock()

		// Send message to request it...
		fmt.Println("Requesting List...")
		SendMessage("GETPEERLIST", "", hostConn)

		// wait until connection list is received
		gotConnsListLock.RLock()
		for !gotConnsList {
			gotConnsListCond.Wait()
			gotConnsListLock.RUnlock()
		}

		// Get the list of all peers from host
		// Get Known names and pk's
		fmt.Println("Requesting Name -> PK map...")
		// ask a known conn for the PK LIST
		connsLock.RLock()
		SendMessage("GETPKMAP", "", conns[len(conns)-1])
		connsLock.RUnlock()

		// wait till received
		gotPkMapLock.RLock()
		for !gotPKmap {
			pkMapCond.Wait()
			gotPkMapLock.RUnlock()
		}
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
	listenerLock.Lock()
	tcpListenerRunning = true
	listenerLock.Unlock()
	listenerCond.Signal()

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
	println("<< type: " + typeString + ", ID: " + myAddress)
	myAddressLock.RUnlock()
	// write msg to all known connections
	connsLock.RLock()
	for i := range conns {
		conns[i].Write([]byte(combinedMsg))
	}
	connsLock.RUnlock()
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
		// Request for list of all peers
		case "GETPEERLIST":
			// Received a request to get a list of known peers. Send list
			SendMessage("PEERS", addresses, conn)
			break
		// List of of all peers
		case "PEERS":
			var peerList []string
			err := json.Unmarshal(marshalledMsg, &peerList)
			if err != nil {
				fmt.Print("Could not unmarshal at PEERS..., " + err.Error())
			}

			addressesLock.Lock()
			addresses = peerList
			addresses = append(addresses, conn.LocalAddr().String())
			addressesLock.Unlock()

			fmt.Println("Disconnection from host...")

			// Disconnect from old host
			SendMessage("DISCONNECT", "", conn)

			connsLock.Lock()
			removeConn(conn)
			connsLock.Unlock()

			// Connect to new peers
			connectToPeers()

			addressesLock.RLock()
			myAddr := addresses[len(addresses)-1]
			addressesLock.RUnlock()

			myAddressLock.Lock()
			myAddress = myAddr
			myAddressLock.Unlock()

			gotConnsListLock.Lock()
			gotConnsList = true
			gotConnsListLock.Unlock()
			gotConnsListCond.Signal()

			// Broadcast that you've connected
			myAddressLock.RLock()
			SendMessageToAll("NEWCONNECTION", myAddress)
			myAddressLock.RUnlock()

			// Ask a peer for previous transactions
			connsLock.RLock()
			peerConn := conns[len(conns)-1]
			connsLock.RUnlock()
			SendMessage("GETALLTRANSACTIONS", "", peerConn)
			break
		// Notification that a new peer has joined the network
		case "NEWCONNECTION":
			var address string
			json.Unmarshal(marshalledMsg, &address)
			// check if the address is already known by this peer
			addressesLock.Lock()
			if !strings.Contains(strings.Join(addresses, ","), address) {
				// Not known, so add to list of addresses
				addresses = append(addresses, address)
			}
			addressesLock.Unlock()
			// Send the message to all the known connections of this peer too
			forward(msgReceived)
			break
		//request all transactions
		case "GETALLTRANSACTIONS":
			var transactions []string
			MessagesSeenLock.RLock()
			for key := range MessagesSeen {
				//Get type of key
				keyType := strings.Split(key, ";")[1]
				//Append all seen transactions to temp
				if keyType == "TRANSACTION" {
					transactions = append(transactions, key)
				}
			}
			MessagesSeenLock.RUnlock()
			SendMessage("ALLTRANSACTIONS", transactions, conn)
			break
		//A list of all transactions that the peer at conn has seen
		case "ALLTRANSACTIONS":
			//unmarshal the array of messages
			var messages []string
			json.Unmarshal(marshalledMsg, &messages)
			for i := range messages {
				//get message
				message := messages[i]

				MessagesSeenLock.Lock()
				//Skip messages already accounted for
				if MessagesSeen[message] {
					MessagesSeenLock.Unlock()
					continue
				}
				MessagesSeen[message] = true
				MessagesSeenLock.Unlock()
				//Get the marshalled transaction
				marshalledTransaction := strings.Split(message, ";")[2]

				//Unmarshal and apply the transaction
				var t account.Transaction
				json.Unmarshal([]byte(marshalledTransaction), &t)
				ledger.Transaction(&t)
			}
			break
		// A transaction
		case "TRANSACTION":
			// Unmarshal the transaction
			var t account.SignedTransaction
			json.Unmarshal(marshalledMsg, &t)
			// Update ledger
			ledger.SignedTransaction(&t)
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
		case "GETPKMAP":
			// received request to get PK map
			SendMessage("PKMAP", ledger.GetPks(), conn)
			break
		case "PKMAP":
			// received the namePK map
			var newPKmap map[string]RSA.PublicKey
			json.Unmarshal(marshalledMsg, &newPKmap)
			ledger.SetPks(newPKmap)

			myName = ledger.EncodePK(myPk)

			gotPkMapLock.Lock()
			gotPKmap = true
			gotPkMapLock.Unlock()
			pkMapCond.Signal()

			// broadcast your pk
			SendMessageToAll("NEWNAMEPK", myPk)
		case "NEWNAMEPK":
			var newPk RSA.PublicKey
			json.Unmarshal(marshalledMsg, &newPk)
			ledger.EncodePK(newPk)
			forward(msgReceived)
		}
	}
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
	fmt.Println("-- sending 100 to each account from my account --")
	// Send 100 to each known peer from your own account.
	for name, _ := range pkMap {
		// skip your own account
		if name == myName {
			continue
		}
		// create a signed transaction
		t := new(account.SignedTransaction)
		t.Amount = 100
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
