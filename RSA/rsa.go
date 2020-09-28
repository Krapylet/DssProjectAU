package RSA

import (
	"math/big"
)

// Opgavebeskrivelsen siger vi skal bruge integers fra math/big pakken

// Public key is (e, n)
// Private key is (d, n)

// e is chosen in the assignment description
var e = 3

func KeyGen(k big.Int) {
	// Choose two primes p and q
	// It must hold for p and q that gcd(e, p-1) = gcd(e, q-1) = 1, where gcd() is greatest common divisor

	// n = p*q   ...  n must have length k
	// d = e^(-1) % (p-1)(q-1)
}

// Encrypt takes som number m that is in the interval [0 .. m-1] and outputs a number c
func Encrypt(m big.Int) big.Int {
	// c = m^e % n
	return *big.NewInt(0)
}

// Decrypt takes some number c and outputs some number m
func Decrypt(c big.Int) big.Int {
	// m = c^d mod n
	return *big.NewInt(0)
}
