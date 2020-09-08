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
	for {
		conn, _ := ln.Accept()
		go receiveMessage(conn)
		go sendMessage(conn)
	}
}

func propagateToOtherThreads(conn net.Conn, MessageSent map[string]bool) {
	for {
		for msg, bool := range MessageSent {
			if !bool {
				bufio.NewWriter(conn).WriteString(msg)
				MessageSent[msg] = true
			}
		}
	}
}

func sendMessage(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		bufio.NewWriter(conn).WriteString(text)
	}
}

func receiveMessage(conn net.Conn) {
	defer conn.Close()
	otherEnd := conn.RemoteAddr().String()
	fmt.Println("Connection established with " + otherEnd)
	var MessageSent map[string]bool
	go propagateToOtherThreads(conn, MessageSent)
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		fmt.Print(msg)
		if err != nil {
			fmt.Println("Ending session with " + otherEnd)
			return
		}
		MessageSent[msg] = false
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
