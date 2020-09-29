package AES

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
)

var iv []byte

/*
MakeAESKey Precondtion: byteLen hhas to be 16, 24 or 32, returns a random key
*/
func MakeAESKey(byteLen int) []byte {

	key := make([]byte, byteLen)
	// fill with random bytes, i.e. make a random key
	_, err := io.ReadFull(rand.Reader, key)

	if err != nil {
		panic(err)
	}

	return key
}

/*
EncryptToFile encrypts message msg with key, writes the output to a file and returns the encrypted message.
The file does not need to exist prior to calling
*/
func EncryptToFile(fileName string, msg []byte, key []byte) {

	// Create a new aes block
	myBlock, _ := aes.NewCipher(key)

	// Ciphertext, should have length equal to the message
	cipherText := make([]byte, len(msg))

	// make the initialization vector and fill it with random bytes
	// the iv most have the same length as the block size
	iv = make([]byte, myBlock.BlockSize())
	_, err := io.ReadFull(rand.Reader, iv)

	if err != nil {
		panic(err)
	}

	// use CTR mode
	stream := cipher.NewCTR(myBlock, iv)

	// Use XOR on the stream
	stream.XORKeyStream(cipherText, msg)

	// write it to file
	writeToFile(fileName, cipherText)
}

/*
DecryptFromFile : Read from file, and decrypt it using the key,
return the byte array containing the decrypted text of the file
*/
func DecryptFromFile(fileName string, key []byte) []byte {
	// get bytes from file
	cipherText := readFromFile(fileName)

	// create msg byte array with same length as cipher text
	msg := make([]byte, len(cipherText))

	// create block with key
	myBlock, _ := aes.NewCipher(key)

	// Setup the counter mode
	stream := cipher.NewCTR(myBlock, iv)

	// XOR
	stream.XORKeyStream(msg, cipherText)

	return msg
}

func writeToFile(fileName string, text []byte) {
	//Create output file
	f, err := os.Create(fileName)

	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	//Write message to file
	_, err = f.Write(text)

	if err != nil {
		panic(err.Error())
	}
}

func readFromFile(fileName string) []byte {
	//read entire file as a bytearray
	text, err := ioutil.ReadFile(fileName)

	if err != nil {
		panic(err)
	}

	return text
}
