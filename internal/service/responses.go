package service

import (
	"github.com/pkg/errors"
)

const (
	OperationSuccessful                = "Operation is successful"
	OperationUnsuccessfulInternalError = "Operation is not successful due to internal error"
	UserNotFound                       = "User with current id wasn't found!"
	InsufficientFunds                  = "Insufficient funds on the balance!"
	DifferentCosts                     = "Order with such parameters has different cost value!"
	OrderNotFound                      = "Order with such parameters wasn't found!"
	InvalidUnmarshalUser               = "Can't unmarshal user from input!"
	InvalidUnmarshalOrder              = "Can't unmarshal order from input!"
	InvalidData                        = "Data don't fit input format!"
	InvalidDate                        = "Invalid data format!"
	AlreadyClosedTransaction           = "Can't get revenue of already closed transaction!"
	OperationOfDifferentUser           = "Operation is bound with different user!"
)

var (
	ErrInsufficientFunds        = errors.New(InsufficientFunds)
	ErrDifferentCosts           = errors.New(DifferentCosts)
	ErrAlreadyClosedTransaction = errors.New(AlreadyClosedTransaction)
	ErrOrderNotFound            = errors.New(OrderNotFound)
	ErrUserNotFound             = errors.New(UserNotFound)
	ErrInvalidDate              = errors.New(InvalidDate)
)

func Wrapf(err error, msg string) error {
	return errors.Wrap(err, msg)
}

type Response struct {
	Error   error       `json:"-"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Balance struct {
	Value uint64 `json:"balance"`
}
