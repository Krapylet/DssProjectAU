package main

import (
	"./RSA"
	"encoding/json"
	"fmt"
	"./peer2peer/account"
	"math/big"
)

func main() {



	pk, sk := RSA.KeyGen(2048)


	fmt.Println(pk)

	toSign := new(account.SignedTransaction)
	toSign.ID = "123:123:" + "1"
	toSign.Amount = 100
	toSign.From = "user-0"
	toSign.To = "user-1"

	fmt.Println(toSign)

	// Create byte array
	m, _ := json.Marshal(&toSign)
	fmt.Println(m)

	// Create big int
	marshToBig := new(big.Int).SetBytes(m)
	fmt.Println(marshToBig)

	// Sign the big int
	s := RSA.Sign(*marshToBig, sk)
	toSign.Signature = s.String()

	// Verify test = toSign
	signature, _ := new(big.Int).SetString(toSign.Signature, 10)
	toSign.Signature = ""



	test := new(account.SignedTransaction)
	test.ID = "123:123:" + "1"
	test.Amount = 100
	test.From = "user-0"
	test.To = "user-1"


	// Create byte array from new struct (removing signature)
	m2, _ := json.Marshal(&test)
	fmt.Println(m2)

	// Create big int of this new struct
	marshToBig2 := new(big.Int).SetBytes(m2)
	fmt.Println(marshToBig2)

	// Verify that signature is equal to new struct, using From as pk
	res := RSA.Verify(*signature, *marshToBig2, pk)
	fmt.Println(res)





}
