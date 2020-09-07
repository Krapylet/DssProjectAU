package main

import (
	"bufio"
	"fmt"
	"net"
)

var MessageSent map[string]bool

func main() {
	c := make(chan string)
	ln, _ := net.Listen("tcp", ":222222") //some random port
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		go handleConnection(conn, c)
	}
}

func handleConnection(conn net.Conn, c chan string) {
	defer conn.Close()
	myEnd := conn.LocalAddr().String()
	otherEnd := conn.RemoteAddr().String()
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Ending session with " + otherEnd)
			return
		}
	}
}
