package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func main() {
	fmt.Println("hej")


	pk, sk := KeyGen(5)

	fmt.Println("Private key, n = ", pk.N, "e = ", pk.E)
	fmt.Println("Secret key, n = ", sk.N, "d = ", sk.D)

	fmt.Println()

	fmt.Println("Encrypt: ")
	m := big.NewInt(404)
	c := Encrypt(pk, m)
	fmt.Println("Cipher is: ", c)

	fmt.Println("Decrypted Cipher: ", Decrypt(sk, c))


}

/*
	RSA: public key consist of two numbers n and e, and the private key consists of two numbers n and d.

	n is 'modulus' and is the product of two prime numbers p,q.
	e most be chosen such that, gcd(e) = gcd(p-1) = gcd(q-1) = 1
	Then d is computed to satisfy: e * d mod (p-1)(q-1) = 1, gives: d = e^(-1) mod (p-1)(q-1)

	To encrypt: c = m^e mod n
	To decrypt: m = c^d mod n
 */

type PublicKey struct {
	N *big.Int
	E *big.Int
}

type SecretKey struct {
	N *big.Int
	D *big.Int
}

// k is the bit length of the generated modulus, n = pq, k=len(n), gives a precondition that k > 3
func KeyGen(k int) (PublicKey, SecretKey) {
	// choose e = 3
	e := big.NewInt(3)

	// Choose p,q as primes such that bitLen(p*q) = k
	// a * b = c, len(c) = len(a+b), if a,b,c is binary numbers

	// Bit length of p, is chosen to be k/2
	lenP := k / 2
	// Length of q, is then len(k)-len(p)
	lenQ := k - lenP

	// choose Primes for p and q
	p , _:= rand.Prime(rand.Reader, lenP)
	q, _ :=  rand.Prime(rand.Reader, lenQ)

	fmt.Println("p is: ", p)
	fmt.Println("q is: ", q)

	// n = p * q
	n := new(big.Int).Mul(p, q)

	fmt.Println("n is: ", n)

	// d is defined as: d = e^(-1) mod (p-1)(q-1)
	// t1 = (p-1)
	t1 := new(big.Int).Sub(p, big.NewInt(1))
	// t2 = (q-1)
	t2 := new(big.Int).Sub(p, big.NewInt(1))
	// t3 = t1*t2 = (p-1)(q-1)
	t3 := new(big.Int).Mul(t1, t2)
	// d = e modinverse t3 = e^(-1) mod (p-1)(q-1)
	d := new(big.Int).ModInverse(e, t3)

	fmt.Println("d is: ", d)

	// Create public and secret key
	pk := new(PublicKey)
	pk.E = e
	pk.N = n

	sk := new(SecretKey)
	sk.D = d
	sk.N = n

	return *pk, *sk
}


func Encrypt(pk PublicKey, m *big.Int) *big.Int {
	// To encrypt: c = m^e mod n

	// calculate m^e
	t := new(big.Int).Exp(m, pk.E, nil)
	// calculate t mod n = m^e mod n
	c := new(big.Int).Mod(t, pk.N)

	return c
}


func Decrypt(sk SecretKey, c *big.Int) *big.Int{
	// To decrypt: m = c^d mod n

	// calculate c^d
	t := new(big.Int).Exp(c, sk.D, nil)
	// Calculate t mod n = c^d mod n
	m := new(big.Int).Mod(t, sk.N)

	return m
}
