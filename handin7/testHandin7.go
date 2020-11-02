package main

import (
	"./RSA"
	"./softwarewallet"
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
)

func main() {
	// automated tests
	testpos()
	testneg()

	// manual test
	// manual()
}

func testpos() {
	fmt.Println("\n--- TESTING-POSITIVE SOFTWAREWALLET ---")
	filename := "testing.txt"
	password := "th3R1ghtP4ssw0rd"
	msg := "TheMsgToSign"

	// returns pk and place AES encrypted with password of SK into 'filename'
	marshalledPK := softwarewallet.Generate(filename, password)
	fmt.Println("AES encrypted the SK using:")
	fmt.Println(" - Filename: " + filename)
	fmt.Println(" - Password: " + password)

	// Unmarshal pk
	var pk RSA.PublicKey
	json.Unmarshal([]byte(marshalledPK), &pk)

	// decrypts from filename with password, and returns the signature of the msg with the decrypted output (SK)
	signature := softwarewallet.Sign(filename, password, []byte(msg))
	fmt.Println("Generated a signature using:")
	fmt.Println(" - Filename: " + filename)
	fmt.Println(" - Password: " + password)

	fmt.Println("Generated_Signature = " + signature[0:10] + "...\n")

	signInt, _ := new(big.Int).SetString(signature, 10)
	msgInt := new(big.Int).SetBytes([]byte(msg))

	fmt.Println("Trying to verify signature with the original msg, using the PK")
	fmt.Println("Verified using pk: ", RSA.Verify(*signInt, *msgInt, pk))
}

func testneg() {
	fmt.Println("\n--- TESTING-NEGATIVE SOFTWAREWALLET ---")
	filename := "testing.txt"
	password := "th3R1ghtP4ssw0rd"
	msg := "TheMsgToSign"

	// returns pk and place AES encrypted with password of SK into 'filename'
	marshalledPK := softwarewallet.Generate(filename, password)
	fmt.Println("AES encrypted the SK using:")
	fmt.Println(" - Filename: " + filename)
	fmt.Println(" - Password: " + password)

	// Unmarshal pk
	var pk RSA.PublicKey
	json.Unmarshal([]byte(marshalledPK), &pk)

	// decrypts from filename with password, and returns the signature of the msg with the decrypted output (SK)
	fmt.Println("Trying to use a wrong password to generate a signature...")
	fmt.Println(" - Filename: " + filename)
	password = "Wr0ngP4ssw0rd"
	fmt.Println(" - Password: " + password)
	softwarewallet.Sign(filename, password, []byte(msg))
}

func manual() {
	fmt.Println("Generate a file...")
	fmt.Print("Filename > ")
	filename, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Print("Password > ")
	password, _ := bufio.NewReader(os.Stdin).ReadString('\n')

	jsonPK := softwarewallet.Generate(filename, password)
	var pk RSA.PublicKey
	json.Unmarshal([]byte(jsonPK), &pk)

	fmt.Println("Generate a signature...")
	fmt.Print("Filename > ")
	filename, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Print("Password > ")
	password, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Print("MsgToSign > ")
	msg, _ := bufio.NewReader(os.Stdin).ReadString('\n')

	signature := softwarewallet.Sign(filename, password, []byte(msg))

	fmt.Println("Generated signature is: " + signature)
}
