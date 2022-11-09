package models

import "time"

type Chain struct {
	ID        int `json:"user_id" example:"1"`
	OrderID   int `json:"order_id" example:"1"`
	ServiceID int `json:"service_id" example:"1"`
}

type Frame struct {
	Chain
	Cost uint64 `json:"cost" example:"100"`
}

type AddBalanceRequest struct {
	ID      int        `json:"user_id" example:"1"`
	Time    *time.Time `json:"time" example:"2020-03-21T12:00:00Z"`
	Comment string     `json:"comment" example:"some description of comment"`
	Balance uint64     `json:"balance" example:"200"`
}

type ReserveRequest struct {
	Frame
	Comment string `json:"comment" example:"some description of comment"`
}

type RevenueRequest struct {
	Frame
	ClosedAt *time.Time `json:"closed_at" example:"2020-03-21T12:00:00Z"`
}
