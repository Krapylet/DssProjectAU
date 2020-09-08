package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Try to connect to the inputted address
	remoteIP, remotePort := inputAddress()
	_, err := net.Dial("tcp", remoteIP+":"+remotePort)
	if err != nil {
		fmt.Println("Could not connect to this address")
	}

	// Print IP and port on this machine
	_, localPort := lookupAddress()

	fmt.Println("Listening...")
	ln, _ := net.Listen("tcp", ":"+localPort) //some random port
	defer ln.Close()

	var MessageSentCollection []map[string]bool
	go sendMessage(MessageSentCollection)
	for {
		var MessageSent map[string]bool
		MessageSentCollection = append(MessageSentCollection, MessageSent)
		conn, _ := ln.Accept()
		go receiveMessage(conn, MessageSentCollection)

	}
}

func propagateToOtherThreads(conn net.Conn, MessageSentCollection []map[string]bool) {
	for {
		for _, MessageSent := range MessageSentCollection {
			for msg, bool := range MessageSent {
				if !bool {
					bufio.NewWriter(conn).WriteString(msg)
					MessageSent[msg] = true
				}
			}
		}

	}
}

func sendMessage(MessageSentCollection []map[string]bool) {

	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		for _, MessageSent := range MessageSentCollection {
			MessageSent[text] = false
		}
	}
}

func receiveMessage(conn net.Conn, MessageSentCollection []map[string]bool) {
	defer conn.Close()
	otherEnd := conn.RemoteAddr().String()
	fmt.Println("Connection established with " + otherEnd)
	go propagateToOtherThreads(conn, MessageSentCollection)
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		fmt.Print(msg)
		if err != nil {
			fmt.Println("Ending session with " + otherEnd)
			return
		}
		for _, MessageSent := range MessageSentCollection {
			MessageSent[msg] = false
		}
	}
}

func inputAddress() (string, string) {
	// Ask for IP and port
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Input IP...")
	remoteIP, _ := reader.ReadString('\n')
	fmt.Println("Input port...")
	remotePort, _ := reader.ReadString('\n')
	remoteIP = strings.TrimSpace(remoteIP)
	remotePort = strings.TrimSpace(remotePort)
	return remoteIP, remotePort
}

func lookupAddress() (string, string) {
	// Display IP and port
	name, _ := os.Hostname()
	address, _ := net.LookupHost(name)
	ip := ""
	port := "18081"
	for _, addr := range address {
		if !strings.Contains(addr, "f") || !strings.Contains(addr, "192.168.") {
			ip = addr
		}
	}
	fmt.Println("Your ip is: " + ip)
	fmt.Println("Your port is: " + port)
	return ip, port
}
