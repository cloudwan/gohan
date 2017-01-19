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
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

const (
	// LongPollHeader is a custom http header to be sent by API's client if he wants to long-poll a resource (using GET). It should contain the latest version Etag. It's used for comparing resource versions and deciding whether to wait for resource to be updated.
	LongPollHeader = "Long-Poll"

	// LongPollEtag is a http header returned by the API along with responses to long-polled GETs. Used mainly for resource versioning for long-polling mechanism, but can also be used for caching.
	LongPollEtag = "Etag"

	longPollPrefix          = "/gohan/long_poll_notifications/"
	longPollNotificationTTL = 10 // sec
)

func calculateResponseEtag(context middleware.Context) string {
	hash := md5.New()
	responseBytes, _ := json.Marshal(context["response"])
	hash.Write(responseBytes)
	etag := fmt.Sprintf(`%x`, hash.Sum(nil))
	log.Debug("[LongPolling] Calculated hash: %s", etag)
	return etag
}

// DbLongPollNotifierWrapper notifies long poll subscribers on modifying DB transactions (Create/Update/Delete) on all HA nodes.
type DbLongPollNotifierWrapper struct {
	db.DB
	gohan_sync.Sync
}

type transactionLongPollNotifier struct {
	transaction.Transaction
	sync         gohan_sync.Sync
	resourcePath string
}

func newTransactionLongPollNotifier(tx transaction.Transaction, sync gohan_sync.Sync) *transactionLongPollNotifier {
	return &transactionLongPollNotifier{tx, sync, ""}
}

// Begin begins a transaction, which will potentially (only for modifying transactions: Create/Update/Delete) notify long poll subscribers on all HA nodes.
func (notifierWrapper *DbLongPollNotifierWrapper) Begin() (transaction.Transaction, error) {
	tx, err := notifierWrapper.DB.Begin()
	if err != nil {
		return nil, err
	}
	return newTransactionLongPollNotifier(tx, notifierWrapper.Sync), nil
}

// Create wraps DB's Create, but also stores path of created resource in structure.
func (notifier *transactionLongPollNotifier) Create(resource *schema.Resource) error {
	if err := notifier.Transaction.Create(resource); err != nil {
		return err
	}
	notifier.resourcePath = resource.Path()
	return nil
}

// Update wraps DB's Update, but also stores path of updated resource in structure.
func (notifier *transactionLongPollNotifier) Update(resource *schema.Resource) error {
	if err := notifier.Transaction.Update(resource); err != nil {
		return err
	}
	notifier.resourcePath = resource.Path()
	return nil
}

// Delete wraps DB's Delete, but also stores path of deleted resource in structure.
func (notifier *transactionLongPollNotifier) Delete(s *schema.Schema, resourceID interface{}) error {
	resource, err := notifier.Fetch(s, transaction.IDFilter(resourceID))
	if err != nil {
		return err
	}
	if err := notifier.Transaction.Delete(s, resourceID); err != nil {
		return err
	}
	notifier.resourcePath = resource.Path()
	return nil
}

// Commit wraps DB's Commit, but also adds a long poll notification entry based on path of resource that was modified in this transaction to etcd so that other HA nodes can react to the change.
func (notifier *transactionLongPollNotifier) Commit() error {
	if err := notifier.Transaction.Commit(); err != nil {
		return err
	}
	if err := AddLongPollNotificationEntry(notifier.resourcePath, notifier.sync); err != nil {
		return err
	}
	return nil
}

// AddLongPollNotificationEntry creates an entry in etcd under longPollPrefix/path/to/resource with a specified time to live.
func AddLongPollNotificationEntry(fullKey string, sync gohan_sync.Sync) error {
	postfix := strings.TrimPrefix(fullKey, statePrefix)
	if postfix == "" {
		// can't long poll on root (path longPollPrefix// is already a directory)
		return nil
	}
	path := longPollPrefix + postfix
	if err := sync.UpdateTTL(path, "dummy", longPollNotificationTTL); err != nil {
		log.Error("[LongPolling] Failed to add long poll notification entry: %s", fullKey)
		return err
	}
	log.Debug("[LongPolling] Added long poll notification entry: %s (TTL = %d sec).", fullKey, longPollNotificationTTL)
	return nil
}

func startLongPollWatchProcess(server *Server) {

	responseChan := make(chan *gohan_sync.Event)
	stopChan := make(chan bool)

	if _, err := server.sync.Fetch(longPollPrefix); err != nil {
		server.sync.Update(longPollPrefix, "")
	}

	go func() {
		defer util.LogFatalPanic(log)
		for server.running {
			err := server.sync.Watch(longPollPrefix, responseChan, stopChan)
			if err != nil {
				log.Error(fmt.Sprintf("sync watch error: %s", err))
			}
		}
	}()
	go func() {
		defer util.LogFatalPanic(log)
		for server.running {
			response := <-responseChan
			go func() {
				// don't notify subs on "expire"
				if response.Action == "create" || response.Action == "set" || response.Action == "update" {
					path := "/" + strings.TrimPrefix(response.Key, longPollPrefix)
					server.longPoll.Broadcast(path)
					log.Debug("[Sync/Long polling] Notified from etcd watch.")
				}
			}()
		}
		stopChan <- true
	}()
}
