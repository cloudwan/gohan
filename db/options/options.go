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
	"github.com/cloudwan/gohan/util"
	"time"
)

// default transaction retry options
const (
	DEFAULT_DEADLOCK_RETRY_TX_INTERVAL = time.Duration(0) * time.Millisecond
	DEFAULT_DEADLOCK_RETRY_TX_COUNT    = 0
)

type Options struct {
	RetryTxCount    int
	RetryTxInterval time.Duration
}

func Read(config *util.Config) Options {
	opts := Options{
		RetryTxCount:    config.GetInt("database/deadlock_retry_tx/count", DEFAULT_DEADLOCK_RETRY_TX_COUNT),
		RetryTxInterval: time.Duration(config.GetInt("database/deadlock_retry_tx/interval_msec", int(DEFAULT_DEADLOCK_RETRY_TX_INTERVAL))) * time.Millisecond,
	}

	if opts.RetryTxCount < 0 {
		panic("database/deadlock_retry_tx/count must not be negative")
	}

	return opts
}

func Default() Options {
	return Options{
		RetryTxCount:    DEFAULT_DEADLOCK_RETRY_TX_COUNT,
		RetryTxInterval: DEFAULT_DEADLOCK_RETRY_TX_INTERVAL,
	}
}
