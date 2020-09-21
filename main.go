package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"./account"
)

// String containing your ip and port "ip:port"
var myAddress string

// All active connections of this peer
var conns []net.Conn

// Addresses of all known peers
var addresses []string

// MessagesSent Set of all messages sent
var MessagesSent = make(map[string]bool)

// Bool to determine if the tcp listener is running
var tcpListenerRunning bool

// Bool to determine if the list of connections is received
var gotConnsList bool

var ledger *account.Ledger = account.MakeLedger()

var transactionCounter = 0

var myPort = ""

func main() {
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
		fmt.Println("Listen for TCP connections at port...")
		myPort, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		myPort = strings.TrimSpace(myPort)

		// Set myAddress
		myAddress = myIP + ":" + myPort

		// add yourself to known peers on the network
		addresses = append(addresses, myIP+":"+myPort)

	} else {
		fmt.Println("Connection Established!")
		defer hostConn.Close()

		// Dont add hostConn to list of active conns since we are disconnecting it soon...

		// Receive message from your host
		go receiveMessage(hostConn)

		// Get the list of all peers from host
		// Send message to request it...
		fmt.Println("Requesting List...")
		sendMessage("!getConnsList\n", hostConn)

		// wait until connection list is received
		for !gotConnsList {
			time.Sleep(time.Second)
		}

		hostConn.Close()
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
		// Prompt for user input and send to all known connections
		msg, _ := bufio.NewReader(os.Stdin).ReadString('\n')

		//_________________DEBUG COMMANDS__________________
		if strings.Contains(msg, "!A") {
			fmt.Println("--- MY ADDRESSES ---")
			fmt.Println("(MY ADDRESS -> " + myAddress + ")")
			for i := range addresses {
				fmt.Println("-> " + addresses[i])
			}
			continue
		}
		if strings.Contains(msg, "!C") {
			fmt.Println("--- MY CONS ---")
			fmt.Println("(MY ADDRESS -> " + myAddress + ")")
			for i := range conns {
				fmt.Println("-> " + conns[i].RemoteAddr().String())
			}
			continue
		}

		if strings.Contains(msg, "!L") {
			fmt.Println("--- MY LEDGER ---")
			for key, value := range ledger.Accounts {
				fmt.Println(key + ": " + strconv.Itoa(value))
			}
			fmt.Println("-----------------")
			continue
		}
		//______________________TRANSACTION COMMAND___________________________
		// "SEND xxxx From YYYY to zzzz"
		var splitMsg []string = strings.Split(msg, " ")
		var isSendCommand bool = splitMsg[0] == "SEND"
		var containsSixArguments bool = len(splitMsg) == 4
		if isSendCommand && containsSixArguments {
			//Convert the command to a transaction object
			var t *account.Transaction = new(account.Transaction)
			t.ID = myAddress + ":" + strconv.Itoa(transactionCounter)
			t.Amount, _ = strconv.Atoi(splitMsg[1])
			t.From = splitMsg[2]
			t.To = splitMsg[3]

			//Apply the transaction locally
			ledger.Transaction(t)

			//overwrite the outgoing message with the marshall of the transaction object
			msg = "!TRANSACTION " + t.ID + " " + strconv.Itoa(t.Amount) + " " + t.From + " " + t.To

			transactionCounter++
		}
		sendMessageToAll(msg)
	}
}

func tcpListener(myIP string, myPort string) {
	// Open own port for incoming TCP
	// Local IP

	ln, err := net.Listen("tcp", ":"+myPort)
	if err != nil {
		fmt.Println("Error listening to: " + myPort)
		panic(-1)
	}
	defer ln.Close()

	// TCP listener is running
	tcpListenerRunning = true

	for {
		fmt.Println("Listening on: " + myIP + ":" + myPort)
		conn, _ := ln.Accept()
		fmt.Println("Got a new connection from: " + conn.RemoteAddr().String())
		// New active connection
		conns = append(conns, conn)
		// New known peer address
		addresses = append(addresses, conn.RemoteAddr().String())

		// Setup message receiver for each new connection
		go receiveMessage(conn)
	}
}

func sendMessageToAll(msg string) {
	// Insert message into map
	MessagesSent[msg] = true
	// write msg to all known connections
	for i := range conns {
		conns[i].Write([]byte(msg))
	}
}

func sendMessage(msg string, conn net.Conn) {
	conn.Write([]byte(msg))
}

func sendListOfPeers(conn net.Conn) {
	// Build string of all known addresses of peers separated by ';'
	peerAddresses := "!PEERS"
	for i := range addresses {
		peerAddresses = peerAddresses + ";" + addresses[i]
	}
	// Send message back to the caller with all known peers.
	fmt.Println("Sending message with all known peers...")
	sendMessage(peerAddresses+"\n", conn)
}

func receiveMessage(conn net.Conn) {
	// Keeps checking for new messages
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println("Error reading message: " + err.Error() + ", disconnecting...")
			return
		}

		// Check if message contains token for request to get list of connections
		if strings.Contains(msg, "!getConnsList") {
			sendListOfPeers(conn)
		}

		// Check if message contains identifier for answer to list of connections
		if strings.Contains(msg, "!PEERS") {
			// Get addresses of peers
			receiveListOfPeers(msg)
			// Disconnect from initial host

			// Connect to 10 newest peers
			connectToPeers()

			// Broadcast that you've connected
			msgToBroadcast := "!NEWCONNECTION;" + myAddress + "\n"
			MessagesSent[msgToBroadcast] = true
			sendMessageToAll(msgToBroadcast)
		}

		// Add to your list of peers
		if strings.Contains(msg, "!NEWCONNECTION") {
			// Check if message has already been received
			_, inMap := MessagesSent[msg]

			// Has NOT been received...
			if !inMap {

				// Insert that this message has been received
				MessagesSent[msg] = true

				// Get the address from the received message
				// Split is 0 = !NEWCONNECTION, 1 = ip:port
				address := strings.Split(msg, ";")[1]
				address = strings.TrimSpace(address)

				// check if the address is already known by this peer
				if !strings.Contains(strings.Join(addresses, ","), address) {
					// Not known, so add to list of addresses
					addresses = append(addresses, address)
				}
				// Send the message to all the known connections of this peer too
				sendMessageToAll(msg)
			}
		}
		// "!TRANSACTION ID AMOUNT FROM TO"
		// A transaction has shape "ID, AMOUNT, FROM, TO"
		if strings.Contains(msg, "!TRANSACTION") {
			if MessagesSent[msg] == true {
				break
			}
			input := strings.Split(msg, " ")
			amount, _ := strconv.Atoi(input[2])
			t := &account.Transaction{input[1], input[3], input[4], amount}
			ledger.Transaction(t)
			sendMessageToAll(msg)
		}

		// Check if the message is contained in the set of messages
		_, inMap := MessagesSent[msg]
		if inMap {
			// msg is contained in map

			// Do nothing ???
		} else {
			// msg is not in map
			// add msg to map
			MessagesSent[msg] = true

			// Print Message
			fmt.Print("[NEW MESSAGE]: " + msg)

			// also send the message to all known connections
			// go sendMessageToAll(msg)
		}
	}
}

func receiveListOfPeers(msg string) {
	fmt.Println("Received List...")
	// Split message at each address, separator is ';'
	msg = strings.TrimSpace(msg)
	splitMsg := strings.Split(msg, ";")
	// Add each peer to Addresses, 1st element is identifier so is skipped
	for i := 1; i < len(splitMsg); i++ {
		fmt.Println("Address added: " + splitMsg[i])
		addresses = append(addresses, splitMsg[i])
	}
	// Set myAddress, is last element in the received list of peers from host.
	myAddress = addresses[len(addresses)-1]
	// List of peers is received
	gotConnsList = true
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
			fmt.Println("Failed to connect to: " + conn.RemoteAddr().String())
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
