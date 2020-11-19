package main

import (
	"./RSA"
	"encoding/json"
	"os"
)

func generate10KeysToFile(filename string) {

	// Text will be:
	// pk:sk;pk:sk;....
	text := ""

	// Generate keys
	for i := 0; i < 10; i++ {
		pk, sk := RSA.KeyGen(2048)
		pkM, _ := json.Marshal(pk)
		skM, _ := json.Marshal(sk)

		text = text + string(pkM) + ";" + string(skM) + ";"
	}

	//Create output file
	f, err := os.Create(filename)

	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	//Write text to file
	_, err = f.Write([]byte(text))

	if err != nil {
		panic(err.Error())
	}
}
