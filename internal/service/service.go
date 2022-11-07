package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	reservation "github.com/antsrp/balance_service/internal/cash_reservation"
	"github.com/antsrp/balance_service/internal/reports"
	"github.com/antsrp/balance_service/internal/user"

	"github.com/antsrp/balance_service/internal/postgres"
)

const (
	REPORTS_RELATIVE_PATH = "../../reports"
)

func getPathToReportsFolder() string {
	curPath, _ := os.Getwd()
	return filepath.Join(curPath, REPORTS_RELATIVE_PATH)
}

type Service struct {
	userStorage        *postgres.UserStorage
	transactionStorage *postgres.TransactionStorage
}

func CreateNewService(us *postgres.UserStorage, ts *postgres.TransactionStorage) *Service {
	return &Service{userStorage: us, transactionStorage: ts}
}

func (s *Service) GetUserBalanceLogic(data string) *Response {
	id, err := strconv.Atoi(data)
	if err != nil {
		return &Response{Error: err, Message: InvalidData}
	}
	resp := &Response{Message: OperationSuccessful}
	if data, err := s.userStorage.GetUserBalance(id); err != nil {
		resp.Error = err
		resp.Message = OperationUnsuccessfulInternalError
	} else {
		resp.Data = Balance{Value: data}
		//resp.Data = fmt.Sprintf(`{"balance": %v}`, data)
	}
	return resp
}

func (s *Service) AddBalanceLogic(data []byte) *Response {
	var u user.User
	if err := json.Unmarshal(data, &u); err != nil {
		return &Response{Error: Wrapf(err, InvalidUnmarshalUser), Message: InvalidData}
	}
	uf, err := s.userStorage.FindUser(u.ID)
	if err != nil {
		return &Response{Error: err, Message: OperationUnsuccessfulInternalError}
	}
	resp := &Response{Message: OperationSuccessful}
	value := u.Balance
	if uf == nil { // no user in base
		if err := s.userStorage.InsertUser(&u); err != nil {
			resp.Error = err
			resp.Message = OperationUnsuccessfulInternalError
		}
	} else { // user was found, update current balance
		u.Balance += uf.Balance
		if err := s.userStorage.UpdateUserBalance(&u); err != nil {
			resp.Error = err
			resp.Message = OperationUnsuccessfulInternalError
		}
	}
	if err := s.transactionStorage.CreateIn(u.ID, u.Time, value, u.Comment); err != nil {
		resp.Error = err
		resp.Message = OperationUnsuccessfulInternalError
	}
	return resp
}

func (s *Service) CashReservationLogic(data []byte) *Response {
	var reserve reservation.CashReservation
	if err := json.Unmarshal(data, &reserve); err != nil {
		return &Response{Error: Wrapf(err, InvalidUnmarshalOrder), Message: InvalidData}
	}
	amount, err := s.transactionStorage.GetAmountOfReservedCash(reserve.UserID)
	if err != nil {
		return &Response{Error: err, Message: OperationUnsuccessfulInternalError}
	}
	balance, err := s.userStorage.GetUserBalance(reserve.UserID)
	if err != nil {
		return &Response{Error: err, Message: OperationUnsuccessfulInternalError}
	}
	if balance < amount+reserve.Cost { // operation is not valid
		return &Response{Error: ErrInsufficientFunds, Message: InsufficientFunds}
	}
	resp := &Response{Message: OperationSuccessful}
	if err := s.transactionStorage.CreateOut(reserve.UserID, reserve.OrderID, reserve.FavorID, reserve.Cost, reserve.Comment); err != nil {
		resp.Error = err
		resp.Message = OperationUnsuccessfulInternalError
	}
	return resp
}

func (s *Service) RevenueLogic(data []byte) *Response {
	var reserve reservation.CashReservation
	if err := json.Unmarshal(data, &reserve); err != nil {
		return &Response{Error: Wrapf(err, InvalidUnmarshalOrder), Message: InvalidData}
	}
	chainID, err := s.transactionStorage.FindTransaction(reserve)

	if err != nil {
		resp := &Response{Message: OperationUnsuccessfulInternalError}

		if err == postgres.ErrClosedTransaction {
			resp.Error = ErrAlreadyClosedTransaction
			resp.Message = AlreadyClosedTransaction
		} else if err == postgres.ErrDifferentCosts {
			resp.Error = ErrDifferentCosts
			resp.Message = ErrDifferentCosts.Error()
		} else if err == postgres.ErrOperationOfDifferentUser {
			resp.Error = ErrOrderNotFound
			resp.Message = OperationOfDifferentUser
		} else if err == postgres.ErrOrderNotFound {
			resp.Error = ErrOrderNotFound
			resp.Message = OrderNotFound
		} else {
			resp.Error = err
		}
		return resp
	}

	i, o := make(chan bool), make(chan bool)
	result := make(chan error)

	resp := &Response{Message: OperationSuccessful}

	go s.userStorage.DecreaseBalanceChained(reserve.UserID, reserve.Cost, i, o, result)
	go s.transactionStorage.CloseTransaction(chainID, reserve.ClosedAt, o, i, result)

	if err1, err2 := <-result, <-result; err1 != nil || err2 != nil {
		if err1 != nil {
			resp.Error = err1
		} else {
			resp.Error = err2
		}
		resp.Message = OperationUnsuccessfulInternalError
	}
	return resp
}

func (s *Service) GetSummaryLogic(year, month int) *Response {
	if (month > 12 || month <= 0) || year <= 0 {
		return &Response{Error: ErrInvalidDate, Message: InvalidDate}
	}
	sum, err := s.transactionStorage.GetMonthSummary(year, month)
	if err != nil {
		return &Response{Error: err, Message: OperationUnsuccessfulInternalError}
	}
	fn, err := reports.WriteToCSV(sum, getPathToReportsFolder())
	if err != nil {
		return &Response{Error: err, Message: OperationUnsuccessfulInternalError}
	}
	return &Response{Message: OperationSuccessful, Data: fn}
}

func (s *Service) GetOperations(user_id, page int, sortby, direction string) *Response {
	operations, err := s.transactionStorage.GetOperations(user_id, page, sortby, direction)
	if err != nil {
		if err == postgres.ErrSortParamNotFound {
			return &Response{Error: err, Message: InvalidData}
		}
		return &Response{Error: err, Message: OperationUnsuccessfulInternalError}
	}
	return &Response{Message: OperationSuccessful, Data: operations}
}
