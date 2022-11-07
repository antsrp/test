package postgres

import (
	"database/sql"

	"github.com/antsrp/balance_service/internal/user"
	"github.com/pkg/errors"
)

type UserStorage struct {
	StatementStorage

	createStmt        *sql.Stmt
	findUserStmt      *sql.Stmt
	findBalanceStmt   *sql.Stmt
	updateBalanceStmt *sql.Stmt
}

var _ user.Storage = &UserStorage{}

const (
	createUserQ        = "INSERT INTO users (id, balance) VALUES ($1, $2) RETURNING id"
	findUserByIDQ      = "SELECT id, balance FROM users WHERE id = $1"
	findUserBalanceQ   = "SELECT balance FROM users WHERE id = $1"
	updateUserBalanceQ = "UPDATE users SET balance = $1 WHERE id = $2"
)

// CreateUserStorage creates new user storage
func CreateUserStorage(d *Dbsql) (*UserStorage, error) {
	s := &UserStorage{StatementStorage: Create(d)}

	stmts := []stmt{
		{Query: createUserQ, Dst: &s.createStmt},
		{Query: findUserBalanceQ, Dst: &s.findBalanceStmt},
		{Query: findUserByIDQ, Dst: &s.findUserStmt},
		{Query: updateUserBalanceQ, Dst: &s.updateBalanceStmt},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, errors.Wrap(err, "can't init statements")
	}

	return s, nil
}

func (s *UserStorage) InsertUser(u *user.User) error {
	var id int
	if err := s.createStmt.QueryRow(&u.ID, &u.Balance).Scan(&id); err != nil {
		return errors.Wrapf(err, "cannot create a new user")
	}
	return nil
}

func (s *UserStorage) UpdateUserBalance(u *user.User) error {
	if _, err := s.updateBalanceStmt.Exec(&u.Balance, &u.ID); err != nil {
		return errors.Wrapf(err, "cannot update balance of user")
	}
	return nil
}

func (s *UserStorage) FindUser(id int) (*user.User, error) {
	var u user.User
	if err := s.findUserStmt.QueryRow(&id).Scan(&u.ID, &u.Balance); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "cannot find a user")
	}
	return &u, nil
}

func (s *UserStorage) GetUserBalance(id int) (uint64, error) {
	var balance uint64
	if err := s.findBalanceStmt.QueryRow(&id).Scan(&balance); err != nil {
		return balance, errors.Wrapf(err, "cannot get balance of user")
	}
	return balance, nil
}

func (s *UserStorage) DecreaseBalanceChained(id int, deductable uint64, in, out chan bool, result chan error) {

	balance, err := s.GetUserBalance(id)
	if err != nil {
		out <- false
		result <- err
		return
	}
	balance -= deductable

	tx, err := s.db.DB.Begin()
	if err != nil {
		out <- false
		result <- err
		return
	}

	if _, err := tx.Stmt(s.updateBalanceStmt).Exec(&balance, &id); err != nil {
		out <- false
		result <- err
		return
	}
	out <- true
	val := <-in

	if !val { // chained part returns an error
		result <- nil
		tx.Rollback()
		return
	}
	tx.Commit()
	result <- nil
}
