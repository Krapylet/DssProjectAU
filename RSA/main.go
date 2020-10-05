package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"time"
)

type PublicKey struct {
	N *big.Int
	E *big.Int
}

type SecretKey struct {
	N *big.Int
	D *big.Int
}

// Precondition for k to be > 3
func KeyGen(k int64) (PublicKey, SecretKey) {
	pk := new(PublicKey)
	sk := new(SecretKey)

	one := big.NewInt(1)

	e := big.NewInt(3)

	var p, q, n, T *big.Int

	// choose new primes p and q, until gcd is 1
	for {
		// choose random length for P, using rand(k-3)+2 (saves at least 2 bits for lenQ)
		lenP, _ := rand.Int(rand.Reader, big.NewInt(k-3))
		lenP = new(big.Int).Add(lenP, big.NewInt(2))

		lenQ := new(big.Int).Sub(big.NewInt(k), lenP)

		// Pick random prime with given length
		lenPInt, _ := strconv.Atoi(lenP.String())
		p, _ = rand.Prime(rand.Reader, lenPInt)

		lenQInt, _ := strconv.Atoi(lenQ.String())
		q, _ = rand.Prime(rand.Reader, lenQInt)

		// p and q cannot be equal
		if p.Cmp(q) == 0 {
			continue
		}

		// (p-1)(q-1)
		T = new(big.Int).Mul(new(big.Int).Sub(p, one), new(big.Int).Sub(q, one))

		// Compute GCD for e and T
		gcd := new(big.Int).GCD(nil, nil, e, T)

		if gcd.String() == "1" {

			n = new(big.Int).Mul(p, q)
			break
		}
	}

	d := new(big.Int).ModInverse(e, T)

	pk.E = e
	pk.N = n

	sk.D = d
	sk.N = n

	return *pk, *sk
}

func Encrypt(pk PublicKey, msg big.Int) *big.Int {
	// c = m^e mod n

	if msg.Cmp(pk.N) != -1 {
		panic("Msg is not in range 0 < msg < n-1")
	}

	cipher := new(big.Int).Exp(&msg, pk.E, pk.N)

	return cipher
}

func Decrypt(sk SecretKey, cipher big.Int) *big.Int {
	// m = c^d mod n
	// Compute msg
	msg := new(big.Int).Exp(&cipher, sk.D, sk.N)

	return msg
}

// --- RSA Signatures ---

func main() {

	pk, sk := KeyGen(2000)

	// 10 kB of random data
	msg := make([]byte, 10000)
	io.ReadFull(rand.Reader, msg)

	// 2. Time for hashing
	start := time.Now()
	// Make hash of msg
	hash := big.NewInt(makeSHA256(msg))
	elapsed := time.Since(start)
	fmt.Println("Time spent hashing: ", elapsed)
	fmt.Println("BitsPerSec = ", 10000/elapsed.Seconds())
	//  Around 315.786.149 bits per sec

	// 3. Time for RSA using the hash value
	start = time.Now()
	s := Sign(*hash, sk)
	elapsed = time.Since(start)
	fmt.Println("Time spent signing (hash): ", elapsed)
	fmt.Println("BitsPerSec = ", 10000/elapsed.Seconds())
	// Around 2.520.726 bits per sec

	// 3.
	msg1 := make([]byte, 1999)
	msg2 := make([]byte, 1999)
	msg3 := make([]byte, 1999)
	msg4 := make([]byte, 1999)
	msg5 := make([]byte, 1999)
	io.ReadFull(rand.Reader, msg1)
	io.ReadFull(rand.Reader, msg2)
	io.ReadFull(rand.Reader, msg3)
	io.ReadFull(rand.Reader, msg4)
	io.ReadFull(rand.Reader, msg5)

	msg1Hex := hex.EncodeToString(msg1)
	msg2Hex := hex.EncodeToString(msg2)
	msg3Hex := hex.EncodeToString(msg3)
	msg4Hex := hex.EncodeToString(msg4)
	msg5Hex := hex.EncodeToString(msg5)

	msg1Int, _ := strconv.ParseInt(msg1Hex, 16, 64)
	msg2Int, _ := strconv.ParseInt(msg2Hex, 16, 64)
	msg3Int, _ := strconv.ParseInt(msg3Hex, 16, 64)
	msg4Int, _ := strconv.ParseInt(msg4Hex, 16, 64)
	msg5Int, _ := strconv.ParseInt(msg5Hex, 16, 64)
	fmt.Println(msg1Int)
	start = time.Now()
	Sign(*big.NewInt(msg1Int), sk)
	Sign(*big.NewInt(msg2Int), sk)
	Sign(*big.NewInt(msg3Int), sk)
	Sign(*big.NewInt(msg4Int), sk)
	Sign(*big.NewInt(msg5Int), sk)
	elapsed = time.Since(start)
	fmt.Println("Time spent signing (no hash): ", elapsed)
	fmt.Println("BitsPerSec = ", 10000/elapsed.Seconds())

	// modify msg
	modMsg := new(big.Int).Add(hash, big.NewInt(1))

	fmt.Println("msg: ", hash)
	fmt.Println("Modified msg: ", modMsg)
	fmt.Println("Signed: ", s)
	fmt.Println("Verified msg: ", Verify(*s, *hash, pk))
	fmt.Println("Verified modMsg: ", Verify(*s, *modMsg, pk))
}

/*
	Sign: Sign the message using s = m^d mod n,
	where msg = original msg, sk = secret key,
	returns the signed message
*/
func Sign(msg big.Int, sk SecretKey) *big.Int {
	s := new(big.Int).Exp(&msg, sk.D, sk.N)
	return s
}

/*
	Verify: Verifies if the msg and signed msg match, i.e. m = s^e mod n,
	where s = signedMsg, msg = original msg, pk = public key,
	returns true if the pk makes the signed msg into msg
	else false.
*/
func Verify(s big.Int, msg big.Int, pk PublicKey) bool {
	// Compute m = s^e mod n
	m := new(big.Int).Exp(&s, pk.E, pk.N)

	// if the original message is equal to the (de)signed message
	// using the PK, the SK and PK most match, and therefore it most be signed by
	// the one the SK of this keypair belongs to
	if m.Cmp(&msg) == 0 {
		return true
	}
	return false
}

func makeSHA256(msg []byte) int64 {
	hash := sha256.Sum256(msg)

	// Convert to Hex
	hashedHex := hex.EncodeToString(hash[:])
	// ParseInt(string, base, bitSize), base is 16 since its hex, and bitSize 64 for int64
	hashedInt, _ := strconv.ParseInt(hashedHex, 16, 64)
	return hashedInt
}
