package account

import (
	"../../RSA"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
)

var pksMap = make(map[string]RSA.PublicKey)

// Ledger type
type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

// MakeLedger initializes an empty ledger
func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

// Transaction type
type Transaction struct {
	ID     string
	From   string
	To     string
	Amount int
}

// Transaction Change the amount in a ledger by the given amount
func (l *Ledger) Transaction(t *Transaction) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.Accounts[t.From] -= t.Amount
	l.Accounts[t.To] += t.Amount
}

type SignedTransaction struct {
	ID        string
	From      string
	To        string
	Amount    int
	Signature string
}

func (l *Ledger) SignedTransaction(t *SignedTransaction) {
	l.lock.Lock()
	defer l.lock.Unlock()

	validSignature := validateSignature(t)
	fmt.Println("Transaction is valid:", validSignature)

	if validSignature {
		if !(l.Accounts[t.From] >= t.Amount) {
			fmt.Println("FAILED: Not enough money")
			return
		}
		l.Accounts[t.From] -= t.Amount
		l.Accounts[t.To] += t.Amount
	}
}

func validateSignature(t *SignedTransaction) bool {
	// save signature from t
	signature, _ := new(big.Int).SetString(t.Signature, 10)
	// remove signature from t
	t.Signature = ""
	// Marshal the transaction
	marshTransaction, _ := json.Marshal(t)
	// convert to big int
	bigIntTransaction := new(big.Int).SetBytes(marshTransaction)

	// Put signature back into t
	t.Signature = signature.String()

	// Verify that using the from's PK, the signature is equal to the marshalled transaction with no signature
	return RSA.Verify(*signature, *bigIntTransaction, decodePK(t.From))
}

func (l *Ledger) EncodePK(pk RSA.PublicKey) string {

	name := RSA.MakeSHA256Hex([]byte(pk.N.String()))

	_, inMap := pksMap[name]
	if !inMap {
		pksMap[name] = pk
	}

	return name
}

func decodePK(name string) RSA.PublicKey {
	if val, inMap := pksMap[name]; inMap {
		return val
	}
	// create a dummy public key with N = 0, E = 0
	temp := new(RSA.PublicKey)
	temp.N = big.NewInt(0)
	temp.E = big.NewInt(0)
	return *temp
}

func (l *Ledger) GetPks() map[string]RSA.PublicKey {
	return pksMap
}

func (l *Ledger) SetPks(newPks map[string]RSA.PublicKey) {
	pksMap = newPks
}
