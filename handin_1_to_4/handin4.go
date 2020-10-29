package main

import (
	"./RSA"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"time"
)

func main() {
	testSignVerify()
	measureHashTime()
	measureSigningTimeHash()
	measureSigningTimeNoHash()
}

func testSignVerify() {
	fmt.Println("--- Testing Sign -> Verify ---")
	// Generate Key Pair
	pk, sk := RSA.KeyGen(2000)
	// Make a new message
	msg := "hello world"
	// SHA256 the msg, returns an int64
	hashMsg := big.NewInt(RSA.MakeSHA256([]byte(msg)))
	fmt.Println("Message is:", hashMsg)
	// Sign the message, uses the secret key
	s := RSA.Sign(*hashMsg, sk)
	// Verify the signed message
	fmt.Println("Message verified:", RSA.Verify(*s, *hashMsg, pk))

	fmt.Println()
	// Modify the message to make sure it rejects
	hashMsg = new(big.Int).Add(hashMsg, big.NewInt(1))
	fmt.Println("Modified the original msg to: ", hashMsg)
	fmt.Println("Message verified: ", RSA.Verify(*s, *hashMsg, pk))

	fmt.Println()
}

func measureHashTime() {
	fmt.Println("--- Measuring Hashing Time ---")

	// Create a 10000 byte, byte array
	msg := make([]byte, 10000)
	// Randomly fill it
	io.ReadFull(rand.Reader, msg)

	// get start time
	start := time.Now()
	// Hash the msg
	RSA.MakeSHA256(msg)
	// get the time since start.
	elapsed := time.Since(start)
	fmt.Println("Time Elapsed during hashing:", elapsed)
	// Compute bits per sec
	fmt.Println("BitsPerSec (hashing)", 10000/elapsed.Seconds())
	fmt.Println()
}

func measureSigningTimeHash() {
	fmt.Println("--- Measure Time of signing with hashing and k=2000 ---")

	_, sk := RSA.KeyGen(2000)

	// Create a 10000 byte, byte array
	msg := make([]byte, 10000)
	// Randomly fill it
	io.ReadFull(rand.Reader, msg)
	// Get hashed msg
	hash := big.NewInt(RSA.MakeSHA256(msg))

	start := time.Now()
	RSA.Sign(*hash, sk)
	elapsed := time.Since(start)

	fmt.Println("Time spent signing (with hashing):", elapsed)
	fmt.Println("BitsPerSec (signing with hashing):", 10000/elapsed.Seconds())
	fmt.Println()
}

func measureSigningTimeNoHash() {
	fmt.Println("--- Measure Time of signing with NO hashing and k=2000 ---")
	// The msg has to be split up in blocks since we can only sign
	// messages of k-1 size.

	_, sk := RSA.KeyGen(2000)

	// List of big ints
	var msgList []big.Int
	// Fill message blocks
	for i := 0; i < 5; i++ {
		// Create a 10000 byte, byte array
		msg := make([]byte, 10000)
		// Randomly fill it
		io.ReadFull(rand.Reader, msg)
		// convert to hex
		hex := hex.EncodeToString(msg)
		// convert to int64
		intMsg, _ := strconv.ParseInt(hex, 16, 64)
		// insert into entireMsg
		msgList = append(msgList, *big.NewInt(intMsg))
	}

	start := time.Now()
	for i := 0; i < 5; i++ {
		RSA.Sign(msgList[i], sk)
	}
	elapsed := time.Since(start)

	fmt.Println("Time spent signing (no hash): ", elapsed)
	fmt.Println("BitsPerSec (signing with hasing): ", 10000/elapsed.Seconds())
}
