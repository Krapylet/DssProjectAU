package main

import (
	"./RSA"
	"./softwarewallet"
	"encoding/json"
	"fmt"
	"math/big"
)

func main() {

	fmt.Println("--- TESTING SOFTWAREWALLET ---")
	filename := "testing.txt"
	password := "th3R1ghtP4ssw0rd"

	msg := "TheMsgToSign"

	// returns pk and place AES encrypted with password of SK into 'filename'
	fmt.Println("Generating PK, SK pair...")
	marshalledPK := softwarewallet.Generate(filename, password)
	fmt.Println("AES encrypted the SK using password and placed it into file: " + filename)

	// Unmarshal pk
	var pk RSA.PublicKey
	json.Unmarshal([]byte(marshalledPK), &pk)

	// decrypts from filename with password, and returns the signature of the msg with the decrypted output (SK)
	signature := softwarewallet.Sign(filename, password, []byte(msg))
	fmt.Println("Generated a signature, using '" + filename + "' and password: '" + password + "'")
	fmt.Println("\nSignature =", signature, "\n")

	signInt, _ := new(big.Int).SetString(signature, 10)
	msgInt := new(big.Int).SetBytes([]byte(msg))

	fmt.Println("Trying to verify signature with the original msg")
	fmt.Println("Verified using pk: ", RSA.Verify(*signInt, *msgInt, pk))

}
