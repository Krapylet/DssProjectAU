package softwarewallet

import (
	"../../AES"
	"../../RSA"
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

/*
	Type of signature - string
*/

// Password - should have byte len 32

func Generate(filename string, password string) string {

	// pwHash32 := []byte(RSA.MakeSHA256Hex([]byte(password)))[0:32]

	// Hash the password 2.000.000 times...
	hashedPW := password
	start := time.Now()
	for i := 0; i < 2000000; i++ {
		hashedPW = RSA.MakeSHA256Hex([]byte(hashedPW))
	}
	elapsed := time.Since(start)
	fmt.Println("elapsed:", elapsed)
	// get first 32 bytes of hashed pw
	pwHash32 := []byte(hashedPW)[0:32]

	// Generate keys
	pk, sk := RSA.KeyGen(2048)

	// Json Marshal the SK
	toEncrypt, err := json.Marshal(sk)
	if err != nil {
		panic(err)
	}

	// Encrypt to file
	AES.EncryptToFile(filename, toEncrypt, pwHash32)

	// return public key, as string "n:e"
	pkAsString := pk.N.String() + ":" + pk.E.String()

	return pkAsString
}

func Sign(filename string, password string, msg []byte) string {

	// Hash the password 50 times...
	hashedPW := password
	for i := 0; i < 2000000; i++ {
		hashedPW = RSA.MakeSHA256Hex([]byte(hashedPW))
	}
	// get first 32 bytes of hashed pw
	pwHash32 := []byte(hashedPW)[0:32]

	// Get SK from file and unmarshal
	jsonSK := AES.DecryptFromFile(filename, pwHash32)
	var sk RSA.SecretKey
	err := json.Unmarshal(jsonSK, &sk)
	if err != nil {
		panic("Wrong password, failed to unmarshal")
	}

	msgToBigInt := new(big.Int).SetBytes(msg)

	signInt := RSA.Sign(*msgToBigInt, sk)

	return signInt.String()
}
