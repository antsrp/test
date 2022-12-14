package postgres

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	_ "github.com/lib/pq" // postgres driver
	"github.com/pkg/errors"
)

type PSQLConfig struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"db"`
	} `yaml:"psql"`
	Limitations struct {
		PageLimit int `yaml:"operations_per_page"`
	} `yaml:"limitations"`
}

// Dbsql struct for connection
type Dbsql struct {
	DB     *sql.DB
	Logger *zap.Logger
	Open   bool
}

func SQLConnect(config *PSQLConfig, log *zap.Logger) (*Dbsql, error) {
	var d Dbsql

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.Database.User, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.DBName)

	fmt.Printf("tries to connect to: %s\n", dsn)

	db, err := sql.Open("postgres", dsn)

	if err != nil {
		return nil, errors.Wrap(err, "can't connect to db")
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, errors.Wrap(err, "can't ping db")
	}

	d.DB = db
	d.Logger = log
	d.Open = true

	return &d, nil
}

// CheckConnection function trying to ping db
func (d *Dbsql) CheckConnection() error {
	if err := d.DB.Ping(); err != nil {
		return errors.Wrap(err, "can't ping db: %s")
	}

	return nil
}

// SQLClose function to close connection
func (d *Dbsql) SQLClose() error {
	err := d.DB.Close()
	if err != nil {
		return fmt.Errorf("cannot close db! %s", err)
	}

	d.Open = false

	return nil
}
