// Copyright (C) 2019 NTT Innovation Institute, Inc.
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

package server

import (
	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
)

//DbSyncWrapper wraps db.DB so it logs events in database on every transaction.
type DbSyncWrapper struct {
	db db.DB
}

func NewDbSyncWrapper(db db.DB) db.DB {
	return &DbSyncWrapper{db}
}

// BeginTx wraps transaction object with sync
func (sw *DbSyncWrapper) BeginTx(options ...transaction.Option) (transaction.Transaction, error) {
	tx, err := sw.db.BeginTx(options...)
	if err != nil {
		return nil, err
	}
	return syncTransactionWrap(tx), nil
}

func (sw *DbSyncWrapper) Connect(dbType string, conn string, maxOpenConn int) error {
	return sw.db.Connect(dbType, conn, maxOpenConn)
}

func (sw *DbSyncWrapper) Close() {
	sw.db.Close()
}

func (sw *DbSyncWrapper) RegisterTable(s *schema.Schema, cascade, migrate bool) error {
	return sw.db.RegisterTable(s, cascade, migrate)
}

func (sw *DbSyncWrapper) DropTable(s *schema.Schema) error {
	return sw.db.DropTable(s)
}

func (sw *DbSyncWrapper) Options() options.Options {
	return sw.db.Options()
}

type transactionEventLogger struct {
	transaction.Transaction
	eventLogged bool
	lastEventId int64
}

func syncTransactionWrap(tx transaction.Transaction) *transactionEventLogger {
	return &transactionEventLogger{tx, false, badEventId}
}
