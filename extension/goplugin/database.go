// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goplugin

import (
	"context"

	gohan_db "github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
)

// Database in an implementation of IDatabase
type Database struct {
	raw  gohan_db.DB
	opts goext.DbOptions
}

// NewDatabase creates new database implementation
func NewDatabase(db gohan_db.DB) *Database {
	opts := db.Options()
	return &Database{
		raw: db,
		opts: goext.DbOptions{
			RetryTxCount:    opts.RetryTxCount,
			RetryTxInterval: opts.RetryTxInterval,
		},
	}
}

// Clone allocates a clone of Database; object may be nil
func (db *Database) Clone() *Database {
	if db == nil {
		return nil
	}
	return &Database{
		raw: db.raw,
	}
}

// Begin starts a new transaction
func (db *Database) Begin() (goext.ITransaction, error) {
	t, _ := db.raw.Begin()
	return &Transaction{t}, nil
}

// BeginTx starts a new transaction with options
func (db *Database) BeginTx(ctx goext.Context, options *goext.TxOptions) (goext.ITransaction, error) {
	opts := transaction.TxOptions{IsolationLevel: transaction.Type(options.IsolationLevel)}
	t, _ := db.raw.BeginTx(context.Background(), &opts)
	return &Transaction{t}, nil
}

// Options return database options rom configuration file
func (db *Database) Options() goext.DbOptions {
	return db.opts
}
