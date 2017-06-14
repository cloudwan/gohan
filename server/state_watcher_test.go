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

package server_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	gohan_sync "github.com/cloudwan/gohan/sync"
)

var _ = Describe("Updating the state", func() {
	const (
		statePrefix      = "/state_watch/state"
		monitoringPrefix = "/state_watch/monitoring"
	)

	var (
		networkSchema   *schema.Schema
		networkResource *schema.Resource
		wrappedTestDB   db.DB
		possibleEvent   gohan_sync.Event
	)

	BeforeEach(func() {
		manager := schema.GetManager()
		var ok bool
		networkSchema, ok = manager.Schema("network")
		Expect(ok).To(BeTrue())
		network := getNetwork("Red", "red")
		var err error
		networkResource, err = manager.LoadResource("network", network)
		Expect(err).ToNot(HaveOccurred())
		wrappedTestDB = &srv.DbSyncWrapper{DB: testDB}
		tx, err := wrappedTestDB.Begin()
		defer tx.Close()
		Expect(err).ToNot(HaveOccurred())
		Expect(tx.Create(networkResource)).To(Succeed())
		Expect(tx.Commit()).To(Succeed())
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

	Describe("Updating state", func() {
		Context("Invoked correctly", func() {
			It("Should work", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())

				tx, err := wrappedTestDB.Begin()
				Expect(err).ToNot(HaveOccurred())
				defer tx.Close()
				afterState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Commit()).To(Succeed())
				Expect(afterState.ConfigVersion).To(Equal(int64(1)))
				Expect(afterState.StateVersion).To(Equal(int64(1)))
				Expect(afterState.State).To(Equal("Ni malvarmetas"))
				Expect(afterState.Error).To(Equal(""))
				Expect(afterState.Monitoring).To(Equal(""))
			})

			It("Should ignore backwards updates", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(0),
						"error":   "",
						"state":   "Ni varmegas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())

				tx, err := wrappedTestDB.Begin()
				Expect(err).ToNot(HaveOccurred())
				defer tx.Close()
				afterState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Commit()).To(Succeed())
				Expect(afterState.ConfigVersion).To(Equal(int64(1)))
				Expect(afterState.StateVersion).To(Equal(int64(1)))
				Expect(afterState.State).To(Equal("Ni malvarmetas"))
				Expect(afterState.Error).To(Equal(""))
				Expect(afterState.Monitoring).To(Equal(""))
			})

			It("Should ignore status updates beyond the most recent config version", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni varmegas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())

				tx, err := wrappedTestDB.Begin()
				Expect(err).ToNot(HaveOccurred())
				defer tx.Close()
				afterState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Commit()).To(Succeed())
				Expect(afterState.ConfigVersion).To(Equal(int64(1)))
				Expect(afterState.StateVersion).To(Equal(int64(1)))
				Expect(afterState.State).To(Equal("Ni malvarmetas"))
				Expect(afterState.Error).To(Equal(""))
				Expect(afterState.Monitoring).To(Equal(""))
			})
		})

		Context("Invoked incorrectly", func() {
			It("With wrong key should return nil", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + strings.Replace(networkResource.Path(), "network", "netwerk", 1),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				err := watcher.StateUpdate(&possibleEvent)
				Expect(err).ToNot(HaveOccurred())
			})

			It("With wrong resource ID should return ErrResourceNotFound", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path() + "malesta",
				}
				watcher := srv.NewStateWatcherFromServer(server)
				err := watcher.StateUpdate(&possibleEvent)
				Expect(err).To(Equal(transaction.ErrResourceNotFound))
			})

			It("Without version should return the proper error", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"error": "",
						"state": "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				err := watcher.StateUpdate(&possibleEvent)
				Expect(err).To(MatchError(ContainSubstring("No version")))
			})
		})
	})

	Context("Updating the monitoring state", func() {
		Context("Invoked correctly", func() {
			It("Should work", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version":    float64(1),
						"monitoring": "Ni rigardas tio",
					},
					Key: monitoringPrefix + networkResource.Path(),
				}
				Expect(watcher.MonitoringUpdate(&possibleEvent)).To(Succeed())

				tx, err := wrappedTestDB.Begin()
				Expect(err).ToNot(HaveOccurred())
				defer tx.Close()
				afterMonitoring, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Commit()).To(Succeed())
				Expect(afterMonitoring.ConfigVersion).To(Equal(int64(1)))
				Expect(afterMonitoring.StateVersion).To(Equal(int64(1)))
				Expect(afterMonitoring.State).To(Equal("Ni malvarmetas"))
				Expect(afterMonitoring.Error).To(Equal(""))
				Expect(afterMonitoring.Monitoring).To(Equal("Ni rigardas tio"))
			})

			It("Should ignore updates if state is not up to date", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version":    float64(1),
						"monitoring": "Ni rigardas tio",
					},
					Key: monitoringPrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.MonitoringUpdate(&possibleEvent)).To(Succeed())

				tx, err := wrappedTestDB.Begin()
				Expect(err).ToNot(HaveOccurred())
				defer tx.Close()
				afterMonitoring, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Commit()).To(Succeed())
				Expect(afterMonitoring.ConfigVersion).To(Equal(int64(1)))
				Expect(afterMonitoring.StateVersion).To(Equal(int64(0)))
				Expect(afterMonitoring.State).To(Equal(""))
				Expect(afterMonitoring.Error).To(Equal(""))
				Expect(afterMonitoring.Monitoring).To(Equal(""))
			})

			It("Should ignore updates if version is not equal to config version", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version":    float64(9999),
						"monitoring": "Ni rigardas tio",
					},
					Key: monitoringPrefix + networkResource.Path(),
				}
				Expect(watcher.MonitoringUpdate(&possibleEvent)).To(Succeed())

				tx, err := wrappedTestDB.Begin()
				Expect(err).ToNot(HaveOccurred())
				defer tx.Close()
				afterMonitoring, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Commit()).To(Succeed())
				Expect(afterMonitoring.ConfigVersion).To(Equal(int64(1)))
				Expect(afterMonitoring.StateVersion).To(Equal(int64(1)))
				Expect(afterMonitoring.State).To(Equal("Ni malvarmetas"))
				Expect(afterMonitoring.Error).To(Equal(""))
				Expect(afterMonitoring.Monitoring).To(Equal(""))
			})
		})

		Context("Invoked incorrectly", func() {
			It("With wrong key should return nil", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version":    float64(1),
						"monitoring": "Ni rigardas tio",
					},
					Key: monitoringPrefix + strings.Replace(networkResource.Path(), "network", "netwerk", 1),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				err := watcher.MonitoringUpdate(&possibleEvent)
				Expect(err).ToNot(HaveOccurred())
			})

			It("With wrong resource ID should return ErrResourceNotFound", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version":    float64(1),
						"monitoring": "Ni rigardas tio",
					},
					Key: monitoringPrefix + networkResource.Path() + "malesta",
				}
				watcher := srv.NewStateWatcherFromServer(server)
				err := watcher.MonitoringUpdate(&possibleEvent)
				Expect(err).To(Equal(transaction.ErrResourceNotFound))
			})

			It("Without monitoring should return the proper error", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
					},
					Key: monitoringPrefix + networkResource.Path(),
				}
				err := watcher.MonitoringUpdate(&possibleEvent)
				Expect(err).To(MatchError(ContainSubstring("No monitoring")))
			})

			It("Without version should return the proper error", func() {
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"version": float64(1),
						"error":   "",
						"state":   "Ni malvarmetas",
					},
					Key: statePrefix + networkResource.Path(),
				}
				watcher := srv.NewStateWatcherFromServer(server)
				Expect(watcher.StateUpdate(&possibleEvent)).To(Succeed())
				possibleEvent = gohan_sync.Event{
					Action: "this is ignored here",
					Data: map[string]interface{}{
						"monitoring": "Ni rigardas tio",
					},
					Key: monitoringPrefix + networkResource.Path(),
				}
				err := watcher.MonitoringUpdate(&possibleEvent)
				Expect(err).To(MatchError(ContainSubstring("No version")))
			})
		})
	})
})
