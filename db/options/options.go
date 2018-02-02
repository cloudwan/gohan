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

package options

import (
	"time"

	"github.com/cloudwan/gohan/util"
)

// default transaction retry options
const (
	DefaultDeadlockRetryTxInterval = time.Duration(0) * time.Millisecond
	DefaultDeadlockRetryTxCount    = 0
)

// Options is type for retry transaction options
type Options struct {
	RetryTxCount    int
	RetryTxInterval time.Duration
}

// Read gets retry transaction options from config
func Read(config *util.Config) Options {
	opts := Options{
		RetryTxCount:    config.GetInt("database/deadlock_retry_tx/count", DefaultDeadlockRetryTxCount),
		RetryTxInterval: time.Duration(config.GetInt("database/deadlock_retry_tx/interval_msec", int(DefaultDeadlockRetryTxInterval))) * time.Millisecond,
	}

	if opts.RetryTxCount < 0 {
		panic("database/deadlock_retry_tx/count must not be negative")
	}

	return opts
}

// Default returns default retry transaction options
func Default() Options {
	return Options{
		RetryTxCount:    DefaultDeadlockRetryTxCount,
		RetryTxInterval: DefaultDeadlockRetryTxInterval,
	}
}
