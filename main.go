package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Known connections for this peer
var conns []net.Conn

// Set of all messages sent
var MessagesSent = make(map[string]bool)

// Bool to determine if the tcp listener is running
var tcpListenerRunning bool

func main() {
	// Try to connect to existing Peer
	// Ask for IP and Port
	fmt.Println("Connect to IP...")
	remoteIP, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	remoteIP = strings.TrimSpace(remoteIP)
	fmt.Println("Connect to port...")
	remotePort, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	remotePort = strings.TrimSpace(remotePort)

	fmt.Println("Trying to connect to: " + remoteIP + ":" + remotePort)
	hostConn, _ := net.Dial("tcp", remoteIP+":"+remotePort)

	if hostConn == nil {
		fmt.Println("No existing peer found at: " + remoteIP + ":" + remotePort)
	} else {
		fmt.Println("Connection Established!")
		defer hostConn.Close()

		// Add Host to known connections
		conns = append(conns, hostConn)
		// also receive message from your host
		go receiveMessage(hostConn)
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
		panic(-1)
	}
	defer ln.Close()

	// TCP listener is running
	tcpListenerRunning = true

	for {
		fmt.Println("Listening on: " + localIP + ":" + localPort)
		conn, _ := ln.Accept()
		fmt.Println("Got a new connection from: " + conn.RemoteAddr().String())
		conns = append(conns, conn)

		// Setup message receiver for each new connection
		go receiveMessage(conn)

		// OPTIONAL: send all previous messages to the new connection
		sendPreviousMessages(conn)
	}
}

func sendPreviousMessages(conn net.Conn) {
	time.Sleep(time.Second * 1)
	for msg, _ := range MessagesSent {
		conn.Write([]byte(msg))
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

func receiveMessage(conn net.Conn) {
	// Keeps checking for new messages
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading message: " + err.Error() + ", disconnecting...")
			return
		}
		// Check if the message is contained in the set of messages
		_, ok := MessagesSent[msg]
		if ok {
			// msg is contained in map

			// Do nothing ???
		} else {
			// msg is not in map
			// add msg to map
			MessagesSent[msg] = true

			// Print Message
			fmt.Print("[NEW MESSAGE]: " + msg)

			// also send the message to all known connections
			go sendMessageToAll(msg)
		}
	}
}
