package main

import (
	"./peer2peer/softwarewallet"
	"./RSA"
	"fmt"
	"math/big"
	"strings"
)

func main() {

	filename := "testing.txt"
	password := "password"

	msg := "heeej"

	// returns pk and place AES encrypted with password of SK into 'filename'
	pk := softwarewallet.Generate(filename, password)

	// 0 = N, 1 = E
	splitPk := strings.Split(pk, ":")

	// decrypts from filename with password, and returns the signature of the msg with the decrypted output (SK)
	signature := softwarewallet.Sign(filename, password, []byte(msg))

	myPK := new(RSA.PublicKey)
	pkN, _ := new(big.Int).SetString(splitPk[0], 10)
	pkE, _ := new(big.Int).SetString(splitPk[1], 10)

	myPK.N = pkN
	myPK.E = pkE


	//fmt.Println("Signature:", signature)
	signInt, _ := new(big.Int).SetString(signature, 10)
	msgInt := new(big.Int).SetBytes([]byte(msg))


	fmt.Println("Verified with pk: ", RSA.Verify(*signInt, *msgInt, *myPK))
}
