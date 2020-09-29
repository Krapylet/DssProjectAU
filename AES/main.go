package main

import (
	"bufio"
	"crypto/aes"
	"fmt"
	"io"
	"os"
)

func main() {
	// Key has to have length 16, 24 or 32
	EncryptToFile("aaaabbbbccccdddd")

}

func EncryptToFile(key string) {
	// Open file
	f, _ := os.Open("./test.txt")
	defer f.Close()
	reader := bufio.NewReader(f)
	buffer := make([]byte, 16)
	block, _ := aes.NewCipher([]byte(key))
	encryptedBytes := make([]byte, 0)
	for {
		_, err := io.ReadFull(reader, buffer)
		if err != nil {
			break
		}
		block.Encrypt(buffer, buffer)
		encryptedBytes = append(encryptedBytes, buffer...)
	}
	fmt.Println(string(encryptedBytes))
}

func DecryptToFile(key string) {
	//block, _ = aes.NewCipher([]byte(key))
}
