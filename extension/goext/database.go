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

package goext

import (
	"strings"
	"time"

	gohan_logger "github.com/cloudwan/gohan/log"
	"fmt"
)

// DbOptions represent database options
type DbOptions struct {
	RetryTxCount    int
	RetryTxInterval time.Duration
}

// IDatabase is an interface to database in Gohan
type IDatabase interface {
	Begin() (ITransaction, error)
	BeginTx(context Context, options *TxOptions) (ITransaction, error)

	Options() DbOptions
}

var log = gohan_logger.NewLogger()

// DefaultDbOptions returns default database options: no retries at all
func DefaultDbOptions() DbOptions {
	return DbOptions{
		RetryTxCount:    0,
		RetryTxInterval: 0,
	}
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

func withinJoinable(tx ITransaction, fn func(tx ITransaction) error) error {
	return fn(tx)
}

func withinDetached(db IDatabase, context Context, txBegin func() (ITransaction, error), fn func(tx ITransaction) error) error {
	opts := db.Options()
	retryTxCount := opts.RetryTxCount
	retryTxInterval := opts.RetryTxInterval
	var tx ITransaction
	var err error

	for attempt := 0; attempt <= retryTxCount; attempt++ {
		tx, err = txBegin()

		if err != nil {
			log.Warning(fmt.Sprintf("failed to begin scoped transaction: %s", err))
			return err
		}

		context["transaction"] = tx

		err = fn(tx)

		if err == nil {
			err = tx.Commit()

			if err == nil {
				delete(context, "transaction")
				return nil
			}
		} else if !tx.Closed() {
			errClose := tx.Close()
			if errClose != nil {
				log.Warning(fmt.Sprintf("close scoped database transaction failed with error: %s", errClose))
			}
		}

		delete(context, "transaction")

		log.Debug("scoped database transaction failed with error: %s", err)

		if !IsDeadlock(err) {
			delete(context, "transaction")
			return err
		}

		log.Warning(fmt.Sprintf("scoped transaction deadlocked, retrying %d / %d", attempt, retryTxCount))
		time.Sleep(retryTxInterval)
	}

	log.Warning(fmt.Sprintf("scoped transaction still deadlocked after %d retries; gave up", retryTxCount))
	return err
}

func within(db IDatabase, context Context, txBegin func() (ITransaction, error), fn func(tx ITransaction) error) error {
	rawTx, joinable := context["transaction"]

	if joinable {
		return withinJoinable(rawTx.(ITransaction), fn)
	}

	return withinDetached(db, context, txBegin, fn)
}

// Within calls a function in scoped transaction
func Within(env IEnvironment, context Context, fn func(tx ITransaction) error) error {
	db := env.Database()
	return within(db, context, func() (ITransaction, error) {
		return db.Begin()
	}, fn)
}

// WithinTx calls a function in scoped transaction with options
func WithinTx(env IEnvironment, context Context, options *TxOptions, fn func(tx ITransaction) error) error {
	db := env.Database()
	return within(db, context, func() (ITransaction, error) {
		return db.BeginTx(context, options)
	}, fn)
}
