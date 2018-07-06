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
)

// DbOptions represent database options
type DbOptions struct {
	RetryTxCount    int
	RetryTxInterval time.Duration
}

// IDatabase is an interface to database in Gohan
type IDatabase interface {
	// Begin starts a new transaction
	Begin() (ITransaction, error)
	// Begin starts a new transaction with options
	BeginTx(context Context, options *TxOptions) (ITransaction, error)

	// Options return database options from the configuration file
	Options() DbOptions

	// Within calls a function in a scoped transaction
	Within(context Context, fn func(tx ITransaction) error) error
	// WithinTx calls a function in a scoped transaction with options
	WithinTx(context Context, options *TxOptions, fn func(tx ITransaction) error) error
}

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
