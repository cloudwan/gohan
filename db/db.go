// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package db

import (
	"math/rand"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
)

//DefaultMaxOpenConn will applied for db object
const DefaultMaxOpenConn = 100

//DB is a common interface for handing db
type DB interface {
	Connect(string, string, int) error
	Close()
	BeginTx(options ...transaction.Option) (transaction.Transaction, error)
	RegisterTable(s *schema.Schema, cascade, migrate bool) error
	DropTable(*schema.Schema) error

	// options
	Options() options.Options
}

// ITransaction is a common interface for transaction
type ITransaction interface {
	Commit() error
	Close() error
	Closed() bool
}

// IsDeadlock checks if error is deadlock
func IsDeadlock(err error) bool {
	knownDatabaseErrorMessages := []string{
		"Deadlock found when trying to get lock; try restarting transaction", /* MySQL / MariaDB */
		"database is locked",                                                 /* SQLite */
	}

	for _, msg := range knownDatabaseErrorMessages {
		if strings.Contains(err.Error(), msg) {
			return true
		}
	}

	return false
}

func tryToCommit(fn func(ITransaction) error) func(ITransaction) error {
	return func(tx ITransaction) error {
		if err := fn(tx); err != nil {
			return err
		}
		return tx.Commit()
	}
}

func tryWithinTx(
	beginStrategy func() (ITransaction, error),
	fn func(ITransaction) error,
) error {
	tx, err := beginStrategy()
	if err != nil {
		log.Warning("failed to begin scoped transaction: %s", err)
		return err
	}

	defer func() {
		if tx != nil && !tx.Closed() {
			if err := tx.Close(); err != nil {
				log.Warning(
					"close scoped database transaction failed with error: %s",
					err,
				)
			}
		}
	}()

	if err = tryToCommit(fn)(tx); err != nil {
		log.Debug("scoped database transaction failed with error: %s", err)
	}
	return err
}

func WithinTemplate(
	retries int,
	retryStrategy func() time.Duration,
	beginStrategy func() (ITransaction, error),
	fn func(ITransaction) error,
) error {
	var err error
	for attempt := 0; attempt <= retries; attempt++ {
		if err = tryWithinTx(beginStrategy, fn); err == nil || !IsDeadlock(err) {
			return err
		}
		retryInterval := GetRetryInterval(retryStrategy())
		log.Warning(
			"scoped transaction deadlocked, retrying %d / %d, after %dms",
			attempt,
			retries,
			retryInterval.Nanoseconds()/int64(time.Millisecond),
		)
		if retryInterval > 0 {
			time.Sleep(retryInterval)
		}
	}
	log.Warning(
		"scoped transaction still deadlocked after %d retries; gave up",
		retries,
	)
	return err
}

func GetRetryInterval(retryInterval time.Duration) time.Duration {
	if retryInterval > 0 {
		// Add random duration between [0, interval] to decrease collision chance
		return retryInterval + time.Duration(rand.Intn(int(retryInterval.Nanoseconds())))
	}
	return 0
}

type FuncInTransaction func(transaction.Transaction) error

// WithinTx executes a scoped transaction with options on a database
func WithinTx(db DB, fn FuncInTransaction, options ...transaction.Option) error {
	return WithinTemplate(db.Options().RetryTxCount,
		func() time.Duration {
			return db.Options().RetryTxInterval
		},
		func() (ITransaction, error) {
			return db.BeginTx(options...)
		},
		func(tx ITransaction) error {
			return fn(tx.(transaction.Transaction))
		},
	)
}

type InitDBParams struct {
	DropOnCreate, Cascade, AutoMigrate, AllowEmpty bool
}

func DefaultTestInitDBParams() InitDBParams {
	return InitDBParams{
		DropOnCreate: true, // always drop DB during tests
		Cascade:      false,
		AutoMigrate:  false, // do not migrate during tests
		AllowEmpty:   true,  // allow tests to run without schemas
	}
}
