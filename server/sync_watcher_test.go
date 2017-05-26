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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
)

const (
	lockPathPrefix    = "/gohan/cluster/lock/watch"
	processPathPrefix = "/gohan/cluster/process"
	masterTTL         = 10
)

var _ = Describe("Sync watcher test", func() {
	BeforeEach(func() {
		watcher := srv.NewSyncWatcherFromServer(server)
		go watcher.Run(context.Background())
		time.Sleep(time.Second)
	})

	AfterEach(func() {
		tx, err := testDB.Begin()
		Expect(err).ToNot(HaveOccurred(), "Failed to create transaction.")
		defer tx.Close()
		for _, schema := range schema.GetManager().Schemas() {
			if whitelist[schema.ID] {
				continue
			}
			err = clearTable(tx, schema)
			Expect(err).ToNot(HaveOccurred(), "Failed to clear table.")
		}
		err = tx.Commit()
		Expect(err).ToNot(HaveOccurred(), "Failed to commit transaction.")
	})

	Describe("Sync watch load balancing with HA", func() {

		It("should be load balanced based on process number", func() {
			// Run as Single Node
			sync := server.GetSync()
			prn, err := sync.Fetch(processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(1))

			time.Sleep(time.Second)

			// Only single node, so all watch pathes are taken with this process
			wrn, err := sync.Fetch(lockPathPrefix + "/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(6))

			// (Simulate) New process joined gohan cluster
			newProcessUUID := "ffffffff-ffff-ffff-ffff-fffffffffffe"
			err = sync.Update(processPathPrefix+"/"+newProcessUUID, newProcessUUID)
			defer sync.Delete(processPathPrefix+"/"+newProcessUUID, false)
			Expect(err).ToNot(HaveOccurred())

			// Now, process watcher detects two processes running
			prn, err = sync.Fetch(processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(2))

			time.Sleep(masterTTL / 2 * time.Second)

			// Watched pathes should be load balanced
			// So, this process started watching half of them based on priority
			wrn, err = sync.Fetch(lockPathPrefix + "/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(3))

			time.Sleep(masterTTL * time.Second)

			// It looks other process failed to start watching.
			// So, this process started watching rest half of them
			wrn, err = sync.Fetch(lockPathPrefix + "/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(6))

			// (Simulate) One more process joined gohan cluster
			newProcessUUID2 := "ffffffff-ffff-ffff-ffff-ffffffffffff"
			err = sync.Update(processPathPrefix+"/"+newProcessUUID2, newProcessUUID2)
			defer sync.Delete(processPathPrefix+"/"+newProcessUUID2, false)
			Expect(err).ToNot(HaveOccurred())

			// Now, process watcher detects three processes running
			prn, err = sync.Fetch(processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(3))

			time.Sleep(masterTTL / 2 * time.Second)

			// Watched pathes are load balanced by 3 processes
			// So, this process started watching two of them based on priority
			wrn, err = sync.Fetch(lockPathPrefix + "/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(2))

			// (Simulate) One process of gohan cluster is down
			err = sync.Delete(processPathPrefix+"/"+newProcessUUID, false)
			Expect(err).ToNot(HaveOccurred())

			// Now, process watcher detects two processes running
			prn, err = sync.Fetch(processPathPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(prn.Children)).To(Equal(2))

			time.Sleep(masterTTL / 2 * time.Second)

			// This process started watching half of them again based on priority
			wrn, err = sync.Fetch(lockPathPrefix + "/watch/key")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(wrn.Children)).To(Equal(3))
		})
	})
})
