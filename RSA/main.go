package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
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

	pk, sk := KeyGen(2048)

	msg := []byte("heeej")

	// Make hash of msg
	hash := big.NewInt(makeSHA256(msg))

	s := Sign(*hash, sk)

	// modify msg
	modMsg := new(big.Int).Add(hash, big.NewInt(1))

	v := Verify(*s, *modMsg, pk)

	fmt.Println("msg: ", hash)
	fmt.Println("Signed: ", s)
	fmt.Println("Verified: ", v)
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
	// using the PK, the SK and PK most match, and therefore the it most be signed by
	// whoever these keys belongs to
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
