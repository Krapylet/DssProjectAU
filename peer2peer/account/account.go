package account

import (
	"sync"
)

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
