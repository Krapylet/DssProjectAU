package softwarewallet

import (
	"../../AES"
	"../../RSA"
	"math/big"
	"strings"
)

/*
	Type of signature - string
 */


// Password - should have byte len 32

func Generate(filename string, password string) string {
	// TODO: noget hashing af password så den får byte len 32,
	// TODO: ved ikke om vi må bruge bcrypt eller vi selv skal implementere noget selv

	// TODO: Tager de 32 første bytes af sha256, skal nok laves om
	pwHash32 := []byte(RSA.MakeSHA256Hex([]byte(password)))[0:32]

	// Generate keys
	pk, sk := RSA.KeyGen(2048)

	// make sk to string "n:d"
	skAsString := sk.N.String() + ":" +sk.D.String()

	// Encrypt to file
	AES.EncryptToFile(filename, []byte (skAsString), pwHash32)

	// return public key, as string "n:e"
	pkAsString := pk.N.String() + ":" + pk.E.String()

	return pkAsString
}


func Sign(filename string, password string, msg []byte) string {

	// takes first 32-bytes of the hashed pw
	pwHash32 := []byte(RSA.MakeSHA256Hex([]byte(password)))[0:32]

	skAsString := string(AES.DecryptFromFile(filename, pwHash32))
	splitSK := strings.Split(string(skAsString), ":")


	// TODO: En eller anden test, der verificerer om det er en gyldig SK, der er blevet decrypted
	skN, err1 := new(big.Int).SetString(splitSK[0], 10)
	skD, err2 := new(big.Int).SetString(splitSK[1], 10)
	if err1 == false && err2 == false {
		panic("Decrypt failed: Wrong password")
	}

	sk := new(RSA.SecretKey)
	sk.N = skN
	sk.D = skD


	msgToBigInt := new(big.Int).SetBytes(msg)

	signInt := RSA.Sign(*msgToBigInt, *sk)

	return signInt.String()
}
