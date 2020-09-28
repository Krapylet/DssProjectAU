package main

import (
	"crypto/rand"
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

func main() {

	pk, sk := KeyGen(2048)

	m := big.NewInt(123)
	fmt.Println("My msg is:", m)

	c := Encrypt(pk, *m)
	fmt.Println("My cipher text is: ", c)

	originalMsg := Decrypt(sk, *c)
	fmt.Println("My original msg is: ", originalMsg)

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

		fmt.Println("p =", p, " q =", q)

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
			fmt.Println("n =", n)
			break
		}
	}
	fmt.Println("Primes found: p =", p, " q =", q)

	d := new(big.Int).ModInverse(e, T)

	fmt.Println("d = ", d)

	pk.E = e
	pk.N = n

	sk.D = d
	sk.N = n

	return *pk, *sk
}

func Encrypt(pk PublicKey, msg big.Int) *big.Int {
	// c = m^e mod n
	fmt.Println("Encrypting...")

	if msg.Cmp(pk.N) != -1 {
		panic("Msg is not in range 0 < msg < n-1")
	}

	cipher := new(big.Int).Exp(&msg, pk.E, pk.N)

	return cipher
}

func Decrypt(sk SecretKey, cipher big.Int) *big.Int {
	// m = c^d mod n
	fmt.Println("Decrypting...")
	// Compute msg
	msg := new(big.Int).Exp(&cipher, sk.D, sk.N)

	return msg
}
