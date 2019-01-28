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

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

var transactionCommitted = make(chan int64, 1024)

const (
	SyncKeyTxCommitted = "/gohan/cluster/sync/tx_committed"
	badEventId         = -1

	defaultRetryDelay = 500 * time.Millisecond
)

func NewTransactionCommitInformer(sync gohan_sync.Sync) *TransactionCommitInformer {
	return &TransactionCommitInformer{
		sync:       sync,
		retryDelay: getRetryDelay(),
	}
}

func getRetryDelay() time.Duration {
	return util.GetConfig().GetDuration("transaction_commit_informer/retry_delay", defaultRetryDelay)
}

type TransactionCommitInformer struct {
	sync       gohan_sync.Sync
	retryDelay time.Duration
}

func (t *TransactionCommitInformer) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	for {
		select {
		case id := <-transactionCommitted:
			t.notify(ctx, id)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *TransactionCommitInformer) notify(ctx context.Context, lastId int64) {
	for {
		lastId = drain(transactionCommitted, lastId)

		err := t.sync.Update(SyncKeyTxCommitted, buildSyncValue(lastId))
		if err == nil {
			return
		}

		log.Error("Failed to notify about committed transaction: %s", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func drain(ch <-chan int64, v int64) int64 {
	for {
		select {
		case v = <-ch:
		default:
			return v
		}
	}
}

func buildSyncValue(id int64) string {
	type syncedEvent struct {
		EventId int64 `json:"event_id"`
	}

	data, err := json.Marshal(syncedEvent{id})
	if err != nil {
		panic(fmt.Sprintf("Can't marshall data: %s", err))
	}

	return string(data)
}
