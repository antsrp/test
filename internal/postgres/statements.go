package postgres

import (
	"database/sql"

	"github.com/pkg/errors"
)

// StatementStorage struct contains struct of connection and statements
type StatementStorage struct {
	db         *Dbsql
	statements []*sql.Stmt
}

// Create new stmt storage
func Create(d *Dbsql) StatementStorage {
	return StatementStorage{db: d}
}

// Close implements io.Closer interface. It is used for close statements (graceful shutdown)
func (s *StatementStorage) Close() error {
	for _, stmt := range s.statements {
		if err := stmt.Close(); err != nil {
			return errors.Wrap(err, "can't close statement")
		}
	}

	return nil
}

type stmt struct {
	Query string
	Dst   **sql.Stmt
}

func (s *StatementStorage) prepareStatement(query string) (*sql.Stmt, error) {
	stmt, err := s.db.DB.Prepare(query)
	if err != nil {
		return nil, errors.Wrapf(err, "can't prepare query %q", query)
	}

	return stmt, nil
}

func (s *StatementStorage) initStatements(statements []stmt) error {
	for i := range statements {
		statement, err := s.prepareStatement(statements[i].Query)
		if err != nil {
			return err
		}

		*statements[i].Dst = statement
		s.statements = append(s.statements, statement)
	}

	return nil
}
