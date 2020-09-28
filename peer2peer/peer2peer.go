package peer2peer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"../account"
)

// String containing your ip and port "ip:port"
var myAddress string

// All active connections of this peer
var conns []net.Conn

// Addresses of all known peers
var addresses []string

// MessagesSeen Set of all messages sent
var MessagesSeen = make(map[string]bool)

// MessagesSeenLock makes sure the map isnt written to and read from at the same time
var MessagesSeenLock = new(sync.RWMutex)

// Bool to determine if the tcp listener is running
var tcpListenerRunning bool

// Bool to determine if the list of connections is received
var gotConnsList bool

var ledger *account.Ledger = account.MakeLedger()

var MessageIDCounter = 0

func start() {
	// Try to connect to existing Peer
	// Ask for IP and Port
	fmt.Println("Connect to IP...")
	//remoteIP, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	//remoteIP = strings.TrimSpace(remoteIP)
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

	} else {
		fmt.Println("Connection Established!")
		defer hostConn.Close()

		// Receive message from your host
		go receiveMessage(hostConn)

		//Before the client has been assigned a popper port, says it has port NEW. Messages from NEW ports are never put into MessagesSeen
		myAddress = "127.0.0.1:NEW"

		// Get the list of all peers from host
		// Send message to request it...
		fmt.Println("Requesting List...")
		SendMessage("GETPEERLIST", "", hostConn)
		// wait until connection list is received
		for !gotConnsList {
			time.Sleep(time.Second)
		}
	}

	// Listen for incoming TCP connections
	// Split, 0 = ip, 1 = port
	splitMyAddr := strings.Split(myAddress, ":")
	go tcpListener(splitMyAddr[0], splitMyAddr[1])

	// Wait for the TCP listener to run
	for !tcpListenerRunning {
		// wait 1 sec
		time.Sleep(time.Second * 1)
	}

	// SendMessage uses the reader which blocks the TCP listener from starting or something :)
	// so the tcp listener has to be started before running this
	for {
		fmt.Print("To make a new transaction, use: SEND 'amount' 'from' 'to' > ")

		// Prompt for user input and send to all known connections
		msg, _ := bufio.NewReader(os.Stdin).ReadString('\n')

		//Trim msg of leading and trailing whitespace
		msg = strings.TrimSpace(msg)

		//_________________DEBUG COMMANDS__________________
		if strings.Contains(msg, "!A") {
			fmt.Println("--- KNOWN LISTENERS ---")
			fmt.Println("(I LISTEN ON: " + myAddress + ")")
			for i := range addresses {
				fmt.Println("-" + addresses[i])
			}
		}
		if strings.Contains(msg, "!C") {
			fmt.Println("--- MY CONS ---")
			for i := range conns {
				fmt.Println("-" + conns[i].RemoteAddr().String())
			}
		}

		if strings.Contains(msg, "!L") {
			fmt.Println("--- MY LEDGER ---")
			for key, value := range ledger.Accounts {
				fmt.Println(key + ": " + strconv.Itoa(value))
			}
			fmt.Println("-----------------")
		}

		var splitMsg []string = strings.Split(msg, " ")

		//______________________TRANSACTION COMMAND___________________________
		// "SEND 'amount' 'from' 'to'"
		var isSendCommand bool = splitMsg[0] == "SEND"
		if isSendCommand {
			//Convert the command to a transaction object
			var t *account.Transaction = new(account.Transaction)
			t.ID = myAddress + ":" + strconv.Itoa(MessageIDCounter)
			t.Amount, _ = strconv.Atoi(splitMsg[1])
			t.From = splitMsg[2]
			t.To = strings.TrimSpace(splitMsg[3])

			//Apply the transaction locally
			ledger.Transaction(t)

			SendMessageToAll("TRANSACTION", t)
		}

		//________________________QUIT COMMAND_______________________________
		// "QUIT"
		var isQuitCommand bool = splitMsg[0] == "QUIT"
		if isQuitCommand {
			println("Quitting")
			SendMessageToAll("DISCONNECT", myAddress)
			break
		}
	}
}

func tcpListener(myIP string, myPort string) {
	// Open own port for incoming TCP
	// Local IP

	ln, err := net.Listen("tcp", ":"+myPort)
	if err != nil {
		fmt.Println("Error listening to: " + myPort)
		panic(err.Error())
	}
	defer ln.Close()

	// TCP listener is running
	tcpListenerRunning = true
	fmt.Println("LISTENING ON PORT -> " + myIP + ":" + myPort)
	for {
		conn, _ := ln.Accept()
		fmt.Println("Got a new connection from: " + conn.RemoteAddr().String())
		// New active connection
		conns = append(conns, conn)
		// Setup message receiver for each new connection
		go receiveMessage(conn)
	}
}

//Sends a new message to known peers. This increases the messageIDCounter
func SendMessageToAll(typeString string, msg interface{}) {
	// Marshall the object that should be sent
	marshalledMsg, _ := json.Marshal(msg)
	// Calculate the message ID
	var id string = myAddress + ":" + strconv.Itoa(MessageIDCounter)
	MessageIDCounter++
	combinedMsg := id + ";" + typeString + ";" + string(marshalledMsg) + "\n"
	// Insert message into map of known messages
	MessagesSeen[combinedMsg] = true
	println("<< type: " + typeString + ", ID: " + myAddress)
	// write msg to all known connections
	for i := range conns {
		conns[i].Write([]byte(combinedMsg))
	}
}

//forwards messages recieved to known peers without chaning it
func forward(msg string) {
	// write msg to all known connections
	for i := range conns {
		conns[i].Write([]byte(msg))
	}
}

//Sends a message to a single peer. Only used for special purposes such as initialization. The same message can be sent multiple times
func SendMessage(typeString string, msg interface{}, conn net.Conn) {
	// Marshall the object that should be sent
	marshalledMsg, _ := json.Marshal(msg)
	// Calculate message ID
	var id string = myAddress + ":" + strconv.Itoa(MessageIDCounter)
	//Prepend S to the message to show that it shouldn't be mapped
	combinedMsg := "S" + id + ";" + typeString + ";" + string(marshalledMsg) + "\n"
	println("<- type: " + typeString + ", to: " + conn.LocalAddr().String() + ", ID: " + myAddress)
	// write msg to target connection
	conn.Write([]byte(combinedMsg))
}

func receiveMessage(conn net.Conn) {
	// Keeps checking for new messages
	for {
		msgReceived, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			//fmt.Println("Error reading message: " + err.Error() + ", disconnecting...")
			return
		}

		splitMsg := strings.Split(msgReceived, ";")
		typeString := splitMsg[1]
		ID := splitMsg[0]
		println("-> type: " + typeString + ", from: " + conn.LocalAddr().String() + ", ID: " + ID)
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
			// Recevied a request to get a list of known peers. Send list
			SendMessage("PEERS", addresses, conn)
			break
		// List of of all peers
		case "PEERS":
			var peerList []string
			err := json.Unmarshal(marshalledMsg, &peerList)
			if err != nil {
				fmt.Print("Could not unmarshal at PEERS..., " + err.Error())
			}

			addresses = peerList

			addresses = append(addresses, conn.LocalAddr().String())

			fmt.Println("Disconnection from host...")

			//Disconnect from old holst
			SendMessage("DISCONNECT", "", conn)
			removeConn(conn)

			//Connect to new peers
			connectToPeers()
			myAddress = addresses[len(addresses)-1]
			gotConnsList = true

			// Broadcast that you've connected
			SendMessageToAll("NEWCONNECTION", myAddress)

			// Ask a peer for previous transactions
			peerConn := conns[len(conns)-1]
			SendMessage("GETALLTRANSACTIONS", "", peerConn)
			break
		// Nitification that a new peer has jouned the network
		case "NEWCONNECTION":
			var address string
			json.Unmarshal(marshalledMsg, &address)

			// check if the address is already known by this peer
			if !strings.Contains(strings.Join(addresses, ","), address) {
				// Not known, so add to list of addresses
				addresses = append(addresses, address)
			}
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
		//a transaction
		case "TRANSACTION":
			// Unmarshal the transaction
			var t account.Transaction
			json.Unmarshal(marshalledMsg, &t)
			// Update ledger
			ledger.Transaction(&t)
			// Broadcast this transaction
			forward(msgReceived)
			break
		//Recieve a disconnect message
		case "DISCONNECT":
			//Remove connection from known connections
			removeConn(conn)

			//Remove adress from known adresses
			var address string
			json.Unmarshal(marshalledMsg, &address)
			removeAddress(address)

			//Close connection
			conn.Close()
			break
		}
	}
}

func connectToPeers() {
	// connect to, up to 10 newest connections, excluding your host
	connCounter := 0
	// Gets the last position in addresses which are not yourself
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
			conns = append(conns, conn)
			// Setup message receiver for this conn
			go receiveMessage(conn)
		}
		// Update connCounter, i.e. connected to a new peer.
		connCounter++
		// Get next position
		pos--
	}
}

//Removes a connection from the global array of connections
func removeConn(conn net.Conn) {
	//Create a temporary holder for valid connections
	var temp []net.Conn
	//Copy all connections except the one we want to remove into temp
	for i := range conns {
		if conns[i] != conn {
			temp = append(temp, conns[i])
		}
	}
	//Overwrite conns with temp
	conns = temp
}

//Removes an adress from the global array of adresses
func removeAddress(address string) {
	//Create a temporary holder for valid connections
	var temp []string
	//Copy all connections except the one we want to remove into temp
	for i := range addresses {
		if addresses[i] != address {
			temp = append(temp, addresses[i])
		}
	}
	//Overwrite conns with temp
	addresses = temp
}
