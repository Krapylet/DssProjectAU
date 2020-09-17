package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Active connections for this peer
var conns []net.Conn

// First connection made to the network
var hostConn net.Conn

// Known adresses of the other peers
var adresses string = "!CONNS"

// MessagesSent : Set of all messages sent
var MessagesSent = make(map[string]bool)

// Bool to determine if the tcp listener is running
var tcpListenerRunning bool

var myAddress string

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
		// add yourself to known adresses
		adresses = adresses + ";" + remoteIP + ":" + remotePort
	} else {
		fmt.Println("Connection Established!")
		defer hostConn.Close()

		// Add Host to known connections
		conns = append(conns, hostConn)

		// also receive message from your host
		go receiveMessage(hostConn)

		// get list of connections from host
		sendMessage("!getConnsList\n", hostConn)

	}

	// Listen for incoming TCP connections
	go tcpListener()

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
		sendMessageToAll(msg)
	}

}

func tcpListener() {
	// Open own port for incoming TCP
	// Local IP
	// Runs on local network
	localIP := "127.0.0.1"
	fmt.Println("Listen for TCP connections at port...")
	localPort, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	localPort = strings.TrimSpace(localPort)

	ln, err := net.Listen("tcp", ":"+localPort)
	if err != nil {
		fmt.Println("Error listening to: " + localPort)
		panic(err)
	}
	defer ln.Close()

	// TCP listener is running
	tcpListenerRunning = true

	for {
		fmt.Println("Listening on: " + localIP + ":" + localPort)
		conn, _ := ln.Accept()
		fmt.Println("Got a new connection from: " + conn.RemoteAddr().String())
		conns = append(conns, conn)
		adresses = adresses + ";" + conn.RemoteAddr().String()

		// Setup message receiver for each new connection
		go receiveMessage(conn)
	}
}

// func sendPreviousMessages(conn net.Conn) {
// 	for msg, _ := range MessagesSent {
// 		time.Sleep(time.Second * 1)
// 		conn.Write([]byte(msg))
// 	}
// }

func sendMessageToAll(msg string) {
	// Insert message into map
	// MessagesSent[msg] = true
	// write msg to all known connections
	for i := range conns {
		conns[i].Write([]byte(msg))
	}
}

func sendMessage(msg string, conn net.Conn) {
	conn.Write([]byte(msg))
}

func receiveMessage(conn net.Conn) {
	// Keeps checking for new messages
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading message: " + err.Error() + ", disconnecting...")
			return
		}
		fmt.Println("MESSAGE AT START: " + msg)
		//Check if the message is contained in the set of messages
		_, ok := MessagesSent[msg]
		if ok {
			// msg is contained in map

			// Do nothing ???
		} else {
			// msg is not in map
			// add msg to map
			MessagesSent[msg] = true

			// connection asks for all connections on the network
			if strings.Contains(msg, "!getConnsList") {
				fmt.Println("HEJ")
				// start message with command identifier
				// Send the string of connections back to the caller

				sendMessage(adresses+"\n", conn)
			} else if strings.Contains(msg, "!CONNS") { // check if the message is answer token to the list of connections
				fmt.Println(msg)
				// Split the string containing the connections into an array
				splitAdresses := strings.Split(msg, ";")
				// Dial up all connections (index 0 is the command name)
				for i := 1; i < len(splitAdresses); i++ {
					adresses = adresses + ";" + splitAdresses[i]
				}
				// Connect to the 10 latest connections other than this one itself (which is the last).
				fmt.Println("SA len: ", len(splitAdresses), "; i: ", len(splitAdresses)-2, "; Guard: ", len(splitAdresses)-12, ";")
				for i := len(splitAdresses) - 2; i > len(splitAdresses)-12 && i >= 0; i-- {
					fmt.Println("i inside loop is: ", i)
					//Do not duplicate listener on host port
					if splitAdresses[i] == hostConn.RemoteAddr().String() {
						continue
					}

					conn, _ := net.Dial("tcp", splitAdresses[i])
					fmt.Println("Connection established to: " + conn.RemoteAddr().String())
					conns = append(conns, conn)

					go receiveMessage(conns[i])
				}
			}

			fmt.Println("Known Adresses is: " + adresses)

			// Print Message
			fmt.Print("[NEW MESSAGE]: " + msg)

			// also send the message to all known connections
			// go sendMessageToAll(msg)
		}
	}
}
