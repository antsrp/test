package reservation

import "time"

type CashReservation struct {
	UserID   int        `json:"user_id"`
	FavorID  int        `json:"service_id"`
	OrderID  int        `json:"order_id"`
	ClosedAt *time.Time `json:"closed_at,omitempty"`
	Comment  string     `json:"comment"`
	Cost     uint64     `json:"cost"`
}
