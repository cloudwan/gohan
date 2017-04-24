package goplugin

import (
	"context"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
)

// Database in an implementation of IDatabase
type Database struct {
	db db.DB
}

// NewDatabase creates new database implementation
func NewDatabase(environment *Environment) goext.IDatabase {
	return &Database{db: environment.db}
}

// Begin starts a new transaction
func (db *Database) Begin() (goext.ITransaction, error) {
	t, _ := db.db.Begin()
	return &Transaction{t}, nil
}

// BeginTx starts a new transaction with options
func (db *Database) BeginTx(ctx goext.Context, options *goext.TxOptions) (goext.ITransaction, error) {
	opts := transaction.TxOptions{IsolationLevel: transaction.Type(options.IsolationLevel)}
	t, _ := db.db.BeginTx(context.Background(), &opts)
	return &Transaction{t}, nil
}
