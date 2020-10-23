package account

import (
	"../../RSA"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
)

var pks []RSA.PublicKey

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
	// Verify that using the from's PK, the signature is equal to the marshalled transaction with no signature

	// Put signature back into t
	t.Signature = signature.String()

	return RSA.Verify(*signature, *bigIntTransaction, decodePK(t.From))
}

func (l *Ledger) EncodePK(pk RSA.PublicKey) string {
	pks = append(pks, pk)
	index := strconv.Itoa(len(pks) - 1)
	return "user-" + index
}

func decodePK(name string) RSA.PublicKey {
	splitName := strings.Split(name, "-")
	index, _ := strconv.Atoi(splitName[1])
	return pks[index]
}

func (l *Ledger) GetPks() []RSA.PublicKey {
	return pks
}

func (l *Ledger) SetPks(newPks []RSA.PublicKey) {
	pks = newPks
}
