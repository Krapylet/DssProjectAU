package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var iv []byte

func main() {
	// Key has to have length 16, 24 or 32
	key := make([]byte, 16)
	// fill with random bytes
	_, err := io.ReadFull(rand.Reader, key)

	if err != nil {
		panic(err)
	}

	fmt.Println("Key is: ", key)

	msg := []byte("hej med dig :)\n")

	fmt.Println("msg: " + string(msg))

	c := EncryptToFile("encrypted.txt", msg, key)

	fmt.Println("cipher: ", string(c))

	m := DecryptFromFile("encrypted.txt", key)

	fmt.Println("original message: " + string(m))

}

/*
EncryptToFile encrypts message msg with key, writes the output to a file and returns the encrypted message.
The file does not need to exist prior to calling
*/
func EncryptToFile(fileName string, msg []byte, key []byte) []byte {

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

	return cipherText
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

/*
func EncryptToFile(FileName string, key string) {
	// Open file
	f, _ := os.Open(FileName)
	defer f.Close()
	reader := bufio.NewReader(f)
	buffer := make([]byte, 16)
	block, _ := aes.NewCipher([]byte(key))
	encryptedBytes := make([]byte, 0)
	for {
		n, err := io.ReadFull(reader, buffer)
		if err != nil {
			if err.Error() == "ErrUnexpectedEOF" && n != 0 {
				buffer = PadToSize(buffer, 16)
			} else {
				println("Error, number of bytes read: ", n)
				println("hey" + err.Error() + string(buffer))
				break
			}
		}

		block.Encrypt(buffer, buffer)
		encryptedBytes = append(encryptedBytes, buffer...)
	}
	fmt.Println(string(encryptedBytes))
}

func DecryptToFile(key string) {
	//block, _ = aes.NewCipher([]byte(key))
}

func PadToSize(message []byte, blockSize int) (paddedMessage []byte) {
	characterDeficit := blockSize - (len(message) % blockSize)
	padding := strings.Repeat("X", characterDeficit)
	return []byte(string(message) + padding)
}
*/
