package RSA

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
)

func Run() {
	fmt.Println("hej")

	k := 12
	pk, sk := KeyGen(k)

	fmt.Println("Private key, n = ", pk.N, "e = ", pk.E)
	fmt.Println("Secret key, n = ", sk.N, "d = ", sk.D)

	if k == pk.N.BitLen() {
		fmt.Println("n has length k: k = " + strconv.Itoa(k) + ", len(n) = " + strconv.Itoa(pk.N.BitLen()))
	} else {
		fmt.Println("n does not have length k: k = " + strconv.Itoa(k) + ", len(n) = " + strconv.Itoa(pk.N.BitLen()))
	}

	fmt.Println("Encrypt: ")
	m := big.NewInt(64)
	fmt.Println("Length of m is: " + strconv.Itoa(m.BitLen()))
	c := Encrypt(pk, m)
	fmt.Println("Cipher is: ", c)

	fmt.Println("Decrypted Cipher: ", Decrypt(sk, c))

}

/*
	RSA: public key consist of two numbers n and e, and the private key consists of two numbers n and d.

	n is 'modulus' and is the product of two prime numbers p,q.
	e most be chosen such that, gcd(e, p-1) = gcd(e, q-1) = 1
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
	p := choosePrimes(lenP, e)
	q := choosePrimes(lenQ, e)

	fmt.Println("p is: ", p)
	fmt.Println("q is: ", q)

	// n = p * q
	n := new(big.Int).Mul(p, q)

	fmt.Println("n is: ", n)

	// d is defined as: d = e^(-1) mod (p-1)(q-1)
	// t1 = (p-1)
	t1 := new(big.Int).Sub(p, big.NewInt(1))
	// t2 = (q-1)
	t2 := new(big.Int).Sub(q, big.NewInt(1))
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
	// m has to be in the interval [0 .. n-1]
	if m.Cmp(new(big.Int).Sub(pk.N, big.NewInt(1))) > 0 {
		fmt.Println("Error: m is too large, cannot encode")
		return big.NewInt(-1)
	}

	// calculate m^e
	t := new(big.Int).Exp(m, pk.E, nil)
	// calculate t mod n = m^e mod n
	c := new(big.Int).Mod(t, pk.N)

	return c
}

func Decrypt(sk SecretKey, c *big.Int) *big.Int {
	// To decrypt: m = c^d mod n

	// calculate c^d
	t := new(big.Int).Exp(c, sk.D, nil)
	// Calculate t mod n = c^d mod n
	m := new(big.Int).Mod(t, sk.N)

	return m
}

// choosePrimes calculates a prime with bit-length len, and a gcd with e of 1. This is a stupid implementation
func choosePrimes(len int, e *big.Int) *big.Int {
	// Choose initial prime
	a, _ := rand.Prime(rand.Reader, len)
	for {
		// Calculate GCD
		b := new(big.Int).Sub(a, big.NewInt(1))
		gcd := new(big.Int).GCD(nil, nil, e, b).Int64()

		if gcd == int64(1) {
			// If gcd is 1, we have a suitable prime
			break
		} else {
			// Otherwise calculate a new prime
			a, _ = rand.Prime(rand.Reader, len)
		}
	}
	return a
}
