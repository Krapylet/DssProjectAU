package main

import (
	"./AES"
	"./RSA"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
)

func main() {
	// runs 10 tests of encryption -> decryption of random numbers, using k = 2048,
	// will panic if a mistake decryption was found
	testRSA()

	//
	testAES()
}

// Do 10 random tests with k = 2048
func testRSA() {
	for i := 0; i < 10; i++ {
		pk, sk := RSA.KeyGen(2048)

		m, _ := rand.Int(rand.Reader, big.NewInt(1000000000000))
		println("Message is: ", m)

		c := RSA.Encrypt(pk, *m)
		println("Encrypted Message is: ", c)

		originalMsg := RSA.Decrypt(sk, *c)
		println("Decrypted Message is: ", m)

		if m.Cmp(originalMsg) != 0 {
			println(m)
			panic("Mistake found")
		}
	}
	println("Done")
}

func testAES() {
	// Generate keys
	pk, sk := RSA.KeyGen(2048)
	AESKey := AES.MakeAESKey(16)

	// RSA encrypt message "42"
	cipher := RSA.Encrypt(pk, *big.NewInt(42))

	// AES encrypt the secret key
	keyByteArray, _ := json.Marshal(sk)
	AES.EncryptToFile("TestRSAKey.txt", keyByteArray, AESKey)

	// Decrypt the file. Decrypted message is written to file
	var RSASecretKey RSA.SecretKey
	json.Unmarshal(AES.DecryptFromFile("TestRSAKey.txt", AESKey), &RSASecretKey)

	// check at key'en kan bruges til RSA decryption
	//read entire file as a bytearray

	// Vi behøves ikke at læse filen, DecryptFromFile returnere alligvel bytearrayet
	//cipherOut, err := ioutil.ReadFile("TestRSAKey.txt")

	//if err != nil {
	//	panic(err)
	//}
	decryptedMSG := RSA.Decrypt(RSASecretKey, *cipher)

	fmt.Println("Decrypted message is: ", decryptedMSG.Int64())
}
