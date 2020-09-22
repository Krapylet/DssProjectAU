package main

import (
	"bufio"
	"encoding/json"
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

// MessagesSeen Set of all messages sent
var MessagesSeen = make(map[string]bool)

// Bool to determine if the tcp listener is running
var tcpListenerRunning bool

// Bool to determine if the list of connections is received
var gotConnsList bool

var ledger *account.Ledger = account.MakeLedger()

var MessageIDCounter = 0

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
		myPort, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		myPort = strings.TrimSpace(myPort)

		// Set myAddress
		myAddress = myIP + ":" + myPort

		// add yourself to known peers on the network
		addresses = append(addresses, myIP+":"+myPort)
		println(myIP + ":" + myPort)

	} else {
		fmt.Println("Connection Established!")
		defer hostConn.Close()

		// Dont add hostConn to list of active conns since we are disconnecting it soon...

		// Receive message from your host
		go receiveMessage(hostConn)

		// Get the list of all peers from host
		// Send message to request it...
		fmt.Println("Requesting List...")
		sendMessage("GETCONNSLIST", "", hostConn)
		// wait until connection list is received
		for !gotConnsList {
			time.Sleep(time.Second)
		}

		// we received the list of addresses so the host is connection is not needed
		// fmt.Println("Disconnection from host...")
		//hostConn.Close()
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

		//_________________DEBUG COMMANDS__________________
		if strings.Contains(msg, "!A") {
			fmt.Println("--- MY ADDRESSES ---")
			fmt.Println("(MY ADDRESS -> " + myAddress + ")")
			for i := range addresses {
				fmt.Println("-> " + addresses[i])
			}
		}
		if strings.Contains(msg, "!C") {
			fmt.Println("--- MY CONS ---")
			fmt.Println("(MY ADDRESS -> " + myAddress + ")")
			for i := range conns {
				fmt.Println("-> " + conns[i].RemoteAddr().String())
			}
		}

		if strings.Contains(msg, "!L") {
			fmt.Println("--- MY LEDGER ---")
			for key, value := range ledger.Accounts {
				fmt.Println(key + ": " + strconv.Itoa(value))
			}
			fmt.Println("-----------------")
		}
		//______________________TRANSACTION COMMAND___________________________
		// "SEND xxxx From YYYY to zzzz"
		var splitMsg []string = strings.Split(msg, " ")
		var isSendCommand bool = splitMsg[0] == "SEND"
		var containsSixArguments bool = len(splitMsg) == 4
		if isSendCommand && containsSixArguments {
			//Convert the command to a transaction object
			var t *account.Transaction = new(account.Transaction)
			t.ID = myAddress + ":" + strconv.Itoa(MessageIDCounter)
			t.Amount, _ = strconv.Atoi(splitMsg[1])
			t.From = splitMsg[2]
			t.To = strings.TrimSpace(splitMsg[3])

			//Apply the transaction locally
			ledger.Transaction(t)

			sendMessageToAll("TRANSACTION", t)

			MessageIDCounter++
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

	for {
		fmt.Println("Listening on: " + myIP + ":" + myPort)
		conn, _ := ln.Accept()
		fmt.Println("Got a new connection from: " + conn.RemoteAddr().String())
		// New active connection
		conns = append(conns, conn)
		// New known peer address
		//addresses = append(addresses, conn.RemoteAddr().String())
		//println(conn.RemoteAddr().String())
		// Setup message receiver for each new connection
		go receiveMessage(conn)
	}
}

func sendMessageToAll(typeString string, msg interface{}) {

	marshalledMsg, _ := json.Marshal(msg)
	var id string = myAddress + ":" + strconv.Itoa(MessageIDCounter)
	combinedMsg := id + ";" + typeString + ";" + string(marshalledMsg) + "\n"
	// Insert message into map
	MessagesSeen[combinedMsg] = true
	// write msg to all known connections
	for i := range conns {
		conns[i].Write([]byte(combinedMsg))
	}
}

func forward(msg string) {
	MessagesSeen[msg] = true
	for i := range conns {
		conns[i].Write([]byte(msg))
	}
}

func sendMessage(typeString string, msg interface{}, conn net.Conn) {
	marshalledMsg, _ := json.Marshal(msg)
	var id string = myAddress + ":" + strconv.Itoa(MessageIDCounter)
	combinedMsg := id + ";" + typeString + ";" + string(marshalledMsg) + "\n"
	conn.Write([]byte(combinedMsg))
}

func receiveMessage(conn net.Conn) {
	// Keeps checking for new messages
	for {
		msgReceived, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println("Error reading message: " + err.Error() + ", disconnecting...")
			return
		}

		//Break if we have already seen the message
		_, seen := MessagesSeen[msgReceived]
		if seen {
			continue
		}

		// Messages have the format id;typeString;msg where msg can have any type
		fmt.Println("MESSAGE HERE : " + msgReceived)
		splitMsg := strings.Split(msgReceived, ";")
		typeString := splitMsg[1]
		marshalledMsg := []byte(splitMsg[2])

		switch typeString {
		case "GETCONNSLIST":
			// Recevied a request to get a list of known peers. Send list
			sendMessage("PEERS", addresses, conn)
			break
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
			sendMessage("DISCONNECT", "", conn)
			removeConn(conn)

			//Connect to new peers
			connectToPeers()
			myAddress = addresses[len(addresses)-1]
			gotConnsList = true

			// Broadcast that you've connected
			sendMessageToAll("NEWCONNECTION", myAddress)
			break
		case "NEWCONNECTION":
			var address string
			json.Unmarshal(marshalledMsg, &address)

			// check if the address is already known by this peer
			if !strings.Contains(strings.Join(addresses, ","), address) {
				// Not known, so add to list of addresses
				addresses = append(addresses, address)
				println(address)
			}
			// Send the message to all the known connections of this peer too
			forward(msgReceived)
			break
		case "TRANSACTION":
			// Unmarshal the transaction
			var t account.Transaction
			json.Unmarshal(marshalledMsg, &t)
			// Update ledger
			ledger.Transaction(&t)
			// Broadcast this transaction
			forward(msgReceived)
			break
		case "DISCONNECT":
			removeConn(conn)
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

func removeConn(conn net.Conn) {
	var temp []net.Conn
	for i := range conns {
		if conns[i] != conn {
			temp = append(temp, conns[i])
		}
	}
	conns = temp
}
