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
	// will panic if a mistake decryption was found
	fmt.Println("Starting RSA test")
	testRSA()

	fmt.Println("\nStarting AES test")
	// Encrypts a RSA secret key, decrypts it again and uses it for RSA decryption
	// creates a files "TestRSAKey.txt" with the encryption of the secret RSA key
	testAES()
}

// run a encryption->decryption test of a random number with k = 2048
func testRSA() {
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

func testAES() {
	// Generate keys
	pk, sk := RSA.KeyGen(2048)
	AESKey := AES.MakeAESKey(16)

	// RSA encrypt message "42"
	msg := big.NewInt(42)
	fmt.Println("Message is: ", msg)
	cipher := RSA.Encrypt(pk, *msg)

	// AES encrypt the secret key
	keyByteArray, _ := json.Marshal(sk)
	AES.EncryptToFile("TestRSAKey.txt", keyByteArray, AESKey)

	// Decrypt the file. Decrypted message is written to file
	var RSASecretKey RSA.SecretKey
	json.Unmarshal(AES.DecryptFromFile("TestRSAKey.txt", AESKey), &RSASecretKey)

	// Use the decrypted secret key to decrypt the RSA cipher
	decryptedMSG := RSA.Decrypt(RSASecretKey, *cipher)

	fmt.Println("Decrypted message is: ", decryptedMSG.Int64())
}
