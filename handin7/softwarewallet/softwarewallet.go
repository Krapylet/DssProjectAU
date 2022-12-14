package softwarewallet

import (
	"../AES"
	"../RSA"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/scrypt"
	"math/big"
)

/*
	Type of signature - string
*/

func Generate(filename string, password string) string {
	hashedPW, _ := scrypt.Key([]byte(password), []byte{}, 1<<18, 8, 1, 32)

	// Generate keys
	pk, sk := RSA.KeyGen(2048)

	// Json Marshal the SK
	toEncrypt, err := json.Marshal(sk)
	if err != nil {
		panic(err)
	}

	// Encrypt to file

	AES.EncryptToFile(filename, toEncrypt, hashedPW)

	// marshal PK
	pkMarshal, _ := json.Marshal(pk)

	return string(pkMarshal)
}

func Sign(filename string, password string, msg []byte) string {

	hashedPW, _ := scrypt.Key([]byte(password), []byte{}, 1<<18, 8, 1, 32)

	// Get SK from file and unmarshal
	jsonSK := AES.DecryptFromFile(filename, hashedPW)
	var sk RSA.SecretKey
	err := json.Unmarshal(jsonSK, &sk)
	if err != nil {
		fmt.Println("Failed to unmarshal: Wrong Password")
		return ""
	}

	msgToBigInt := new(big.Int).SetBytes(msg)

	signInt := RSA.Sign(*msgToBigInt, sk)

	return signInt.String()
}
