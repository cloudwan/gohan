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

package server_test

import (
	"context"
	"sort"
	"strings"
	sync_lib "sync"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/sync"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	lockPathPrefix    = "/gohan/cluster/lock/watch"
	processPathPrefix = "/gohan/cluster/process"
	masterTTL         = 10
)

var _ = Describe("Sync watcher test", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		done   sync_lib.WaitGroup
	)

	BeforeEach(func() {
		// go vet complains about cancel(), but it's called in AfterEach
		ctx, cancel = context.WithCancel(context.Background())
		watcher := srv.NewSyncWatcherFromServer(server)
		done.Add(1)
		go func() {
			defer GinkgoRecover()
			Expect(watcher.Run(ctx, &done)).To(Equal(context.Canceled))
		}()

		time.Sleep(time.Second)
	})

	AfterEach(func() {
		Expect(db.WithinTx(testDB, func(tx transaction.Transaction) error {
			for _, schema := range schema.GetManager().Schemas() {
				if whitelist[schema.ID] {
					continue
				}
				Expect(dbutil.ClearTable(ctx, tx, schema)).ToNot(HaveOccurred(), "Failed to clear table.")
			}
			return nil
		})).ToNot(HaveOccurred(), "Failed to create or commit transaction.")
		cancel()
		done.Wait()
	})

	Describe("Sync watch load balancing with HA", func() {

		PIt("should be load balanced based on process number", func() {
			// Run as Single Node
			sync := server.GetSync()
			prn, err := sync.Fetch(ctx, processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(1))

			time.Sleep(time.Second)

			// Only single node, so all watch pathes are taken with this process
			wrn, err := sync.Fetch(ctx, lockPathPrefix+"/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(6))

			// (Simulate) New process joined gohan cluster
			newProcessUUID := "ffffffff-ffff-ffff-ffff-fffffffffffe"
			err = sync.Update(ctx, processPathPrefix+"/"+newProcessUUID, newProcessUUID)
			defer sync.Delete(ctx, processPathPrefix+"/"+newProcessUUID, false)
			Expect(err).ToNot(HaveOccurred())

			// Now, process watcher detects two processes running
			prn, err = sync.Fetch(ctx, processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(2))

			time.Sleep(masterTTL / 2 * time.Second)

			// Watched pathes should be load balanced
			// So, this process started watching half of them based on priority
			wrn, err = sync.Fetch(ctx, lockPathPrefix+"/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(3))

			time.Sleep(masterTTL * time.Second)

			// It looks other process failed to start watching.
			// So, this process started watching rest half of them
			wrn, err = sync.Fetch(ctx, lockPathPrefix+"/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(6))

			// (Simulate) One more process joined gohan cluster
			newProcessUUID2 := "ffffffff-ffff-ffff-ffff-ffffffffffff"
			err = sync.Update(ctx, processPathPrefix+"/"+newProcessUUID2, newProcessUUID2)
			defer sync.Delete(ctx, processPathPrefix+"/"+newProcessUUID2, false)
			Expect(err).ToNot(HaveOccurred())

			// Now, process watcher detects three processes running
			prn, err = sync.Fetch(ctx, processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(3))

			time.Sleep(masterTTL / 2 * time.Second)

			// Watched pathes are load balanced by 3 processes
			// So, this process started watching two of them based on priority
			wrn, err = sync.Fetch(ctx, lockPathPrefix+"/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(2))

			// (Simulate) One process of gohan cluster is down
			err = sync.Delete(ctx, processPathPrefix+"/"+newProcessUUID, false)
			Expect(err).ToNot(HaveOccurred())

			// Now, process watcher detects two processes running
			prn, err = sync.Fetch(ctx, processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(2))

			time.Sleep(masterTTL / 2 * time.Second)

			// This process started watching half of them again based on priority
			wrn, err = sync.Fetch(ctx, lockPathPrefix+"/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(3))
		})
	})

	Describe("Sync watch job handling", func() {

		AfterEach(func() {
			sync := server.GetSync()
			sync.Delete(ctx, "/test/watcher", true)
			sync.Delete(ctx, "/watch/key", true)
		})

		It("should process sequentially for each watch key", func() {
			sync := server.GetSync()

			// this test extension will write watched key (e.g., '/watch/key/1/apple') as a value
			// to a path '/test/<milisec_timestamp>' then sleep 5 seconds.
			// Therefore, sorting fetched results by key means the order of called sync update extensions
			watchKey1a := "/watch/key/1/apple"
			err := sync.Update(ctx, watchKey1a, `{"pref" : "watcher"}`)
			Expect(err).ToNot(HaveOccurred())
			watchKey2a := "/watch/key/2/apple"
			err = sync.Update(ctx, watchKey2a, `{"pref" : "watcher"}`)
			Expect(err).ToNot(HaveOccurred())
			watchKey1b := "/watch/key/1/banana"
			err = sync.Update(ctx, watchKey1b, `{"pref" : "watcher"}`)
			Expect(err).ToNot(HaveOccurred())
			watchKey1c := "/watch/key/1/cherry"
			err = sync.Update(ctx, watchKey1c, `{"pref" : "watcher"}`)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(1 * time.Second)

			wrn, err := sync.Fetch(ctx, "/test/watcher")
			Expect(err).ToNot(HaveOccurred())
			// 2 sync updates for path /watch/key/1/apple and /watch/key/2/apple are ongoing
			Expect(len(wrn.Children)).To(Equal(2))

			sort.Sort(ByValue(wrn.Children))

			childrenValues := []string{wrn.Children[0].Value, wrn.Children[1].Value}
			Expect(childrenValues).To(Equal([]string{watchKey1a, watchKey2a}))

			time.Sleep(5 * time.Second)
			// after 5 sec, another job for /watch/key/1/banana should start

			wrn, err = sync.Fetch(ctx, "/test/watcher")
			Expect(err).ToNot(HaveOccurred())

			sort.Sort(ByKey(wrn.Children))
			Expect(len(wrn.Children)).To(Equal(3))
			Expect(wrn.Children[2].Value).To(Equal(watchKey1b))

			time.Sleep(5 * time.Second)
			// after 5 sec, another job for /watch/key/1/cherry should start

			wrn, err = sync.Fetch(ctx, "/test/watcher")
			Expect(err).ToNot(HaveOccurred())

			sort.Sort(ByKey(wrn.Children))
			Expect(len(wrn.Children)).To(Equal(4))
			Expect(wrn.Children[3].Value).To(Equal(watchKey1c))
		})
	})
})

type ByKey []*sync.Node

func (n ByKey) Len() int           { return len(n) }
func (n ByKey) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByKey) Less(i, j int) bool { return strings.Compare(n[i].Key, n[j].Key) < 0 }

type ByValue []*sync.Node

func (n ByValue) Len() int           { return len(n) }
func (n ByValue) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByValue) Less(i, j int) bool { return strings.Compare(n[i].Value, n[j].Value) < 0 }
