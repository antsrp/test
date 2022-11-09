package user

import "time"

type User struct {
	ID      int        `json:"user_id"`
	Time    *time.Time `json:"time"`
	Comment string     `json:"comment,omitempty"`
	Balance uint64     `json:"balance"`
}

type Storage interface {
	InsertUser(*User) error
	FindUser(id int) (*User, error)
	GetUserBalance(id int) (uint64, error)
	UpdateUserBalance(*User) error
	DeleteAllUsers() error
}
