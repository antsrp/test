package reservation

import "time"

type Transaction struct {
	ID          int
	UserID      int
	Direction   string
	IsCompleted bool
	ChainID     int
	ClosedAt    *time.Time
	Cost        uint64
	Comment     string
}

func NewTransaction(id, user_id int, direction string, cost uint64, comment string) *Transaction {
	return &Transaction{
		ID:          id,
		UserID:      user_id,
		Direction:   direction,
		IsCompleted: false,
		ChainID:     -1,
		ClosedAt:    nil,
		Cost:        cost,
		Comment:     comment,
	}
}

type Storage interface {
	//Create(int, int, int, uint64) error
	CreateIn(int, *time.Time, uint64, string) error
	CreateOut(int, int, int, uint64, string) error
	GetAmountOfReservedCash(int) (uint64, error)
	FindTransaction(CashReservation) (int, error)
	CloseTransaction(int, *time.Time, chan bool, chan bool, chan error)
	DeleteAllTransactions() error
}
