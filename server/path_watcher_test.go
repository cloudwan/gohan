// Copyright (C) 2019 NTT Innovation Institute, Inc.
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
	"fmt"
	"sync"

	"github.com/cloudwan/gohan/extension"
	mock_extension "github.com/cloudwan/gohan/extension/mocks"
	srv "github.com/cloudwan/gohan/server"
	gohan_sync "github.com/cloudwan/gohan/sync"
	mock_sync "github.com/cloudwan/gohan/sync/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Sync watcher test", func() {
	const watchedKey = "/path_watcher/test"

	var (
		ctrl *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	It("Runs registered extensions", func() {
		calledCh := make(chan struct{}, 1)

		mockEnv := mock_extension.NewMockEnvironment(ctrl)
		mockEnv.EXPECT().Clone().Return(mockEnv)
		mockEnv.EXPECT().HandleEvent("notification", gomock.Any()).DoAndReturn(func(interface{}, interface{}) error {
			calledCh <- struct{}{}
			return nil
		})

		wg := sync.WaitGroup{}
		defer wg.Wait()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		Expect(server.GetSync().Delete(ctx, watchedKey, true)).To(Succeed())
		Expect(server.GetSync().Update(ctx, watchedKey, "{}")).To(Succeed())

		pw := srv.NewPathWatcher(
			server.GetSync(),
			map[string]extension.Environment{
				"path_watcher/test": mockEnv,
			},
			watchedKey,
			0,
		)

		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			pw.Run(ctx, &wg)
		}()

		Eventually(calledCh).Should(Receive())
	})

	shouldReceiveExactlyOnce := func(ch <-chan struct{}) {
		Eventually(ch).Should(Receive(), "the channel did not receive anything")
		Consistently(ch).ShouldNot(Receive(), "the channel received twice")
	}

	setupSyncMock := func(watchCh chan *gohan_sync.Event, lockLostCh chan struct{}) *mock_sync.MockSync {
		mockSync := mock_sync.NewMockSync(ctrl)

		mockSync.EXPECT().Lock(gomock.Any(), gomock.Any(), gomock.Any()).Return(lockLostCh, nil)
		mockSync.EXPECT().Unlock(gomock.Any(), gomock.Any()).Return(nil)
		mockSync.EXPECT().Fetch(gomock.Any(), "/gohan/watch/revision"+watchedKey).
			Return(nil, fmt.Errorf("Key not found"))
		mockSync.EXPECT().Watch(gomock.Any(), watchedKey, gomock.Any()).
			DoAndReturn(func(ctx context.Context, _, _ interface{}) <-chan *gohan_sync.Event {
				go func() {
					select {
					case <-ctx.Done():
						close(watchCh)
					}
				}()
				return watchCh
			})
		mockSync.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		return mockSync
	}

	It("Stops processing further events when ETCD lock is lost", func() {
		// simulate two events pending in etcd
		watchCh := make(chan *gohan_sync.Event, 2)
		for i := 0; i < cap(watchCh); i++ {
			watchCh <- &gohan_sync.Event{
				Key: watchedKey,
			}
		}

		lockLostCh := make(chan struct{}, 1)
		mockSync := setupSyncMock(watchCh, lockLostCh)

		calledCh := make(chan struct{}, 1)
		lockClosed := false
		mockEnv := mock_extension.NewMockEnvironment(ctrl)
		mockEnv.EXPECT().Clone().Return(mockEnv).AnyTimes()
		mockEnv.EXPECT().HandleEvent("notification", gomock.Any()).DoAndReturn(func(interface{}, interface{}) error {
			calledCh <- struct{}{}

			// simulate etcd lock is lost when first event is processing
			if !lockClosed {
				close(lockLostCh)
				lockClosed = true
			}
			return nil
		}).AnyTimes()

		wg := sync.WaitGroup{}
		defer wg.Wait()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pw := srv.NewPathWatcher(
			mockSync,
			map[string]extension.Environment{
				"path_watcher/test": mockEnv,
			},
			watchedKey,
			0,
		)

		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			pw.Run(ctx, &wg)
		}()

		shouldReceiveExactlyOnce(calledCh)
	})
})
