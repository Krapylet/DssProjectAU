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
	ip, port := inputAddress()
	_, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		fmt.Println("Could not connect to this address")
	}

	// Print IP and port on this machine
	lookupAddress()

	ln, _ := net.Listen("tcp", ":"+port) //some random port
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		go handleConnection(conn)
	}
}

func sendMessages(conn net.Conn, MessageSent map[string]bool) {
	for {
		for msg, bool := range MessageSent {
			if !bool {
				bufio.NewWriter(conn).WriteString(msg)
				MessageSent[msg] = true
			}
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	otherEnd := conn.RemoteAddr().String()
	var MessageSent map[string]bool
	go sendMessages(conn, MessageSent)
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
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

func lookupAddress() {
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
}
