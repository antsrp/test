package postgres

import (
	"database/sql"
	"strings"
	"time"

	reservation "github.com/antsrp/balance_service/internal/cash_reservation"
	"github.com/antsrp/balance_service/internal/reports"
	"github.com/pkg/errors"
)

const (
	getAmountOfReservedCashQ = "SELECT SUM(cost) FROM transactions WHERE user_id = $1 AND direction = 'out' AND is_completed = false GROUP BY(user_id)"
	createChainQ             = "INSERT INTO chains (order_id, service_id) VALUES ($1, $2) RETURNING id;"
	findChainQ               = "SELECT id FROM chains WHERE order_id = $1 AND service_id = $2"
	createInQ                = "INSERT INTO transactions (user_id, direction, is_completed, closed_at, cost, comment) VALUES ($1, 'in', true, $2, $3, $4);"
	createOutQ               = "INSERT INTO transactions (user_id, direction, is_completed, chain_id, cost, comment) VALUES ($1, 'out', false, $2, $3, $4);"
	findTransactionQ         = "SELECT user_id, is_completed, cost FROM transactions WHERE chain_id = $1"
	updateTransactionQ       = `UPDATE transactions 
	SET closed_at = $1, is_completed = true
	WHERE chain_id = $2`

	summaryOfMonthQ = `SELECT favors.name, SUM(cost)
	FROM transactions
	LEFT JOIN chains ON chain_id = chains.id
	JOIN favors ON chains.service_id = favors.id
	WHERE direction = 'out' AND $1 <= closed_at AND closed_at < $2
	GROUP BY service_id, favors.name;`

	operationsCarcassQ = `SELECT direction, favors.name, cost, comment, closed_at 
	FROM transactions 
	LEFT JOIN chains ON chain_id = chains.id
	LEFT JOIN favors ON chains.service_id = favors.id
	WHERE user_id = $1 AND is_completed = true
	`

	limitsQ = ` LIMIT $2 OFFSET $3`

	operationsDefaultWPagesQ    = operationsCarcassQ + limitsQ
	operationsByDateDESCQ       = operationsCarcassQ + ORDER_BY_DATE + SORT_DESC
	operationsByDateASCQ        = operationsCarcassQ + ORDER_BY_DATE + SORT_ASC
	operationsByCostDESCQ       = operationsCarcassQ + ORDER_BY_SUM + SORT_DESC
	operationsByCostASCQ        = operationsCarcassQ + ORDER_BY_SUM + SORT_ASC
	operationsByDateWPagesDESCQ = operationsByDateDESCQ + limitsQ
	operationsByDateWPagesASCQ  = operationsByDateASCQ + limitsQ
	operationsByCostWPagesDESCQ = operationsByCostDESCQ + limitsQ
	operationsByCostWPagesASCQ  = operationsByCostASCQ + limitsQ

	ClosedTransaction = "Transaction is already closed"
	DifferentCosts    = "Different costs"
	OrderNotFound     = "Wrong order"
	SortParamNotFound = "Wrong sorting param"

	SORT_ASC      = `ASC`
	SORT_DESC     = `DESC`
	SORT_DATE     = `date`
	SORT_SUM      = `sum`
	ORDER_BY_SUM  = ` ORDER BY cost `
	ORDER_BY_DATE = ` ORDER BY closed_at `
)

var (
	ErrClosedTransaction        = errors.New(ClosedTransaction)
	ErrDifferentCosts           = errors.New(DifferentCosts)
	ErrOrderNotFound            = errors.New(OrderNotFound)
	ErrSortParamNotFound        = errors.New(SortParamNotFound)
	ErrOperationOfDifferentUser = errors.New(OrderNotFound)
)

type TransactionStorage struct {
	StatementStorage

	getAmountOfReservedCashStmt  *sql.Stmt
	createChainStmt              *sql.Stmt
	findChainStmt                *sql.Stmt
	createInStmt                 *sql.Stmt
	createOutStmt                *sql.Stmt
	findTransactionStmt          *sql.Stmt
	updateTransactionStmt        *sql.Stmt
	getMonthSummaryStmt          *sql.Stmt
	operationsDefaultStmt        *sql.Stmt
	operationsDefaultWPagesStmt  *sql.Stmt
	operationsDateDescStmt       *sql.Stmt
	operationsDateAscStmt        *sql.Stmt
	operationsCostDescStmt       *sql.Stmt
	operationsCostAscStmt        *sql.Stmt
	operationsDateWPagesDescStmt *sql.Stmt
	operationsDateWPagesAscStmt  *sql.Stmt
	operationsCostWPagesDescStmt *sql.Stmt
	operationsCostWPagesAscStmt  *sql.Stmt

	pageLimit int
}

func CreateTransactionStorage(d *Dbsql, limit int) (*TransactionStorage, error) {
	s := &TransactionStorage{StatementStorage: Create(d)}

	stmts := []stmt{
		{Query: getAmountOfReservedCashQ, Dst: &s.getAmountOfReservedCashStmt},
		{Query: findChainQ, Dst: &s.findChainStmt},
		{Query: createChainQ, Dst: &s.createChainStmt},
		{Query: createInQ, Dst: &s.createInStmt},
		{Query: createOutQ, Dst: &s.createOutStmt},
		{Query: findTransactionQ, Dst: &s.findTransactionStmt},
		{Query: updateTransactionQ, Dst: &s.updateTransactionStmt},
		{Query: summaryOfMonthQ, Dst: &s.getMonthSummaryStmt},
		{Query: operationsCarcassQ, Dst: &s.operationsDefaultStmt},
		{Query: operationsByDateDESCQ, Dst: &s.operationsDateDescStmt},
		{Query: operationsByDateASCQ, Dst: &s.operationsDateAscStmt},
		{Query: operationsByCostDESCQ, Dst: &s.operationsCostDescStmt},
		{Query: operationsByCostASCQ, Dst: &s.operationsCostAscStmt},
		{Query: operationsByDateWPagesDESCQ, Dst: &s.operationsDateWPagesDescStmt},
		{Query: operationsDefaultWPagesQ, Dst: &s.operationsDefaultWPagesStmt},
		{Query: operationsByDateWPagesASCQ, Dst: &s.operationsDateWPagesAscStmt},
		{Query: operationsByCostWPagesDESCQ, Dst: &s.operationsCostWPagesDescStmt},
		{Query: operationsByCostWPagesASCQ, Dst: &s.operationsCostWPagesAscStmt},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, errors.Wrap(err, "can't init statements")
	}

	s.pageLimit = limit

	return s, nil
}

var _ reservation.Storage = &TransactionStorage{}

func (s *TransactionStorage) CreateIn(user_id int, at *time.Time, value uint64, comment string) error {
	c := sql.NullString{String: comment, Valid: comment != ""}
	if _, err := s.createInStmt.Exec(&user_id, &at, &value, &c); err != nil {
		return errors.Wrap(err, "can't create input transaction")
	}
	return nil
}

func (s *TransactionStorage) CreateOut(user_id, order_id, favor_id int, cost uint64, comment string) error {
	var chainID int

	tx, err := s.db.DB.Begin()
	if err != nil {
		return errors.Wrap(err, "can't create a transaction")
	}

	if err := tx.Stmt(s.createChainStmt).QueryRow(&order_id, &favor_id).Scan(&chainID); err != nil {
		tx.Rollback()
		return errors.Wrap(err, "can't create chain of order_id & service_id")
	}

	c := sql.NullString{String: comment, Valid: comment != ""}
	if _, err := tx.Stmt(s.createOutStmt).Exec(&user_id, &chainID, &cost, &c); err != nil {
		tx.Rollback()
		return errors.Wrap(err, "can't create output transaction")
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return errors.Wrap(err, "can't commit transaction")
	}
	return nil
}
func (s *TransactionStorage) GetAmountOfReservedCash(user_id int) (uint64, error) {

	var amount uint64
	if _, err := s.getAmountOfReservedCashStmt.Exec(&user_id); err != nil {
		return 0, errors.Wrap(err, "can't get an amount of reserved cash")
	}

	return amount, nil
}

func (s *TransactionStorage) getTransactionData(chain_id int) (*reservation.Transaction, error) {
	var transaction reservation.Transaction
	if err := s.findTransactionStmt.QueryRow(&chain_id).Scan(&transaction.UserID, &transaction.IsCompleted, &transaction.Cost); err != nil {
		return nil, errors.Wrap(err, "can't get transaction status and cost")
	}
	return &transaction, nil
}

func (s *TransactionStorage) findChainID(orderID, serviceID int) (int, error) {
	var id int
	if err := s.findChainStmt.QueryRow(&orderID, &serviceID).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return -1, ErrOrderNotFound
		}
		return -1, errors.Wrap(err, "can't find a chain id")
	}
	return id, nil
}

func (s *TransactionStorage) FindTransaction(data reservation.CashReservation) (int, error) {
	chainID, err := s.findChainID(data.OrderID, data.FavorID)
	if err != nil {
		return -1, err
	}
	td, err := s.getTransactionData(chainID)
	if err != nil {
		return -1, err
	}
	if td.IsCompleted { // closed already
		return -1, ErrClosedTransaction
	}
	if td.UserID != data.UserID {
		return -1, ErrOperationOfDifferentUser
	}
	if td.Cost != data.Cost {
		return -1, ErrDifferentCosts
	}
	return chainID, nil
}
func (s *TransactionStorage) CloseTransaction(chainID int, closeTime *time.Time, in, out chan bool, result chan error) {
	val := <-in
	if !val { // chained part returns an error
		result <- nil
		return
	}

	tx, err := s.db.DB.Begin()
	if err != nil {
		out <- false
		result <- err
		return
	}

	if _, err := tx.Stmt(s.updateTransactionStmt).Exec(closeTime, &chainID); err != nil {
		tx.Rollback()
		out <- false
		result <- err
		return
	}
	out <- true
	tx.Commit()
	result <- nil
}

func (s *TransactionStorage) GetMonthSummary(year, month int) ([]reports.SummaryCSV, error) {

	begin := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := begin.AddDate(0, 1, 0)

	rows, err := s.getMonthSummaryStmt.Query(begin, end)
	if err != nil {
		return nil, errors.Wrap(err, "can't get summary of month")
	}
	var sum []reports.SummaryCSV
	for rows.Next() {
		var s reports.SummaryCSV
		if err := rows.Scan(&s.Name, &s.Value); err != nil {
			return nil, errors.Wrap(err, "can't get row of month summary")
		}
		sum = append(sum, s)
	}
	return sum, nil
}

func (s *TransactionStorage) getOperationsDefault(stmt *sql.Stmt, user_id int) ([]reports.Operation, error) {
	rows, err := stmt.Query(&user_id)
	if err != nil {
		return nil, errors.Wrap(err, "can't get operations with such parameters")
	}

	var ops []reports.Operation

	for rows.Next() {
		var o reports.Operation
		var comm, favor sql.NullString
		if err := rows.Scan(&o.Type, &favor, &o.Sum, &comm, &o.Time); err != nil {
			return nil, errors.Wrap(err, "can't scan operation row")
		}
		if favor.Valid {
			o.Favor = favor.String
		}
		if comm.Valid {
			o.Comment = comm.String
		}
		ops = append(ops, o)
	}

	return ops, nil
}

func (s *TransactionStorage) GetOperations(user_id, page int, sortby, direction string) ([]reports.Operation, error) {
	sortby, direction = strings.ToLower(sortby), strings.ToUpper(direction)

	if page == 0 && sortby == "" { // default query
		return s.getOperationsDefault(s.operationsDefaultStmt, user_id)
	}

	var offset int
	var stmt *sql.Stmt

	if sortby == "" {
		stmt = s.operationsDefaultWPagesStmt
	} else if sortby == SORT_DATE {
		if page > 0 {
			stmt = s.operationsDateWPagesAscStmt
			if direction == SORT_DESC {
				stmt = s.operationsDateWPagesDescStmt
			}
		} else {
			stmt = s.operationsDateAscStmt
			if direction == SORT_DESC {
				stmt = s.operationsDateDescStmt
			}
			return s.getOperationsDefault(stmt, user_id)
		}
	} else if sortby == SORT_SUM {
		if page > 0 {
			stmt = s.operationsCostWPagesAscStmt
			if direction == SORT_DESC {
				stmt = s.operationsCostWPagesDescStmt
			}
		} else {
			stmt = s.operationsCostAscStmt
			if direction == SORT_DESC {
				stmt = s.operationsCostDescStmt
			}
			return s.getOperationsDefault(stmt, user_id)
		}
	} else {
		return nil, ErrSortParamNotFound
	}

	if page > 0 {
		offset = (page - 1) * s.pageLimit
	}

	rows, err := stmt.Query(&user_id, &s.pageLimit, &offset)
	if err != nil {
		return nil, errors.Wrap(err, "can't get operations with such parameters")
	}

	var ops []reports.Operation

	for rows.Next() {
		var o reports.Operation
		var comm, favor sql.NullString
		if err := rows.Scan(&o.Type, &favor, &o.Sum, &comm, &o.Time); err != nil {
			return nil, errors.Wrap(err, "can't scan operation row")
		}
		if favor.Valid {
			o.Favor = favor.String
		}
		if comm.Valid {
			o.Comment = comm.String
		}
		ops = append(ops, o)
	}

	return ops, nil
}
