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
	"strconv"
	"sync"
	"time"

	"github.com/cloudwan/gohan/extension"
	mock_extension "github.com/cloudwan/gohan/extension/mocks"
	srv "github.com/cloudwan/gohan/server"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sync watcher test", func() {
	const watchedKey = "/path_watcher/test"

	var (
		ctrl    *gomock.Controller
		ctx     context.Context
		pw      *srv.PathWatcher
		mockEnv *mock_extension.MockEnvironment
		wg      *sync.WaitGroup
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.Background()
		wg = &sync.WaitGroup{}

		Expect(server.GetSync().Delete(ctx, "/", true)).To(Succeed())

		mockEnv = mock_extension.NewMockEnvironment(ctrl)

		pw = srv.NewPathWatcher(
			server.GetSync(),
			map[string]extension.Environment{
				"path_watcher/test": mockEnv,
			},
			watchedKey,
			0,
		)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	getRevision := func(path string) int64 {
		node, err := server.GetSync().Fetch(ctx, path)
		Expect(err).NotTo(HaveOccurred())
		return node.Revision
	}

	givenWatchedKeySet := func(value string) int64 {
		Expect(server.GetSync().Update(ctx, watchedKey, value)).To(Succeed())
		return getRevision(watchedKey)
	}

	givenLastSeenRevisionForcedTo := func(revision int64) {
		Expect(server.GetSync().Update(ctx, srv.SyncWatchRevisionPrefix+watchedKey, strconv.FormatInt(revision, 10))).To(Succeed())
	}

	givenEventHandler := func(fn func()) {
		mockEnv.EXPECT().Clone().AnyTimes().Return(mockEnv)
		mockEnv.EXPECT().HandleEvent("notification", gomock.Any()).AnyTimes().DoAndReturn(func(interface{}, interface{}) error {
			fn()
			return nil
		})
	}

	whenPathWatcherStarted := func(ctx context.Context) {
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			pw.Run(ctx, wg)
		}()
	}

	thenExactlyOneEventProcessed := func(ch <-chan struct{}) {
		Eventually(ch).Should(Receive(), "the channel did not receive anything")
		Consistently(ch, time.Second).ShouldNot(Receive(), "the channel received twice")
	}

	thenWatchedKeyUpdatedTo := func(expectedRevision int64, expectedValue string) {
		Eventually(func() (int64, error) {
			node, err := server.GetSync().Fetch(ctx, watchedKey)
			return node.Revision, err
		}).Should(Equal(expectedRevision))

		node, err := server.GetSync().Fetch(ctx, watchedKey)
		Expect(err).NotTo(HaveOccurred())
		Expect(node.Value).To(Equal(expectedValue))
	}

	synchronizeTest := func() (context.Context, func()) {
		testCtx, cancel := context.WithCancel(context.Background())
		return testCtx, func() {
			cancel()
			wg.Wait()
		}
	}

	It("Runs registered extensions", func() {
		testCtx, waitForDone := synchronizeTest()
		defer waitForDone()

		calledCh := make(chan struct{}, 1)
		givenEventHandler(func() {
			calledCh <- struct{}{}
		})

		revision := givenWatchedKeySet("{}")
		givenLastSeenRevisionForcedTo(revision - 1)

		whenPathWatcherStarted(testCtx)

		thenExactlyOneEventProcessed(calledCh)
	})

	It("Stops processing when conflict detected", func() {
		testCtx, waitForDone := synchronizeTest()
		defer waitForDone()

		firstEventRevision := givenWatchedKeySet(`{"index": 1}`)
		givenWatchedKeySet(`{"index": 2}`)
		givenWatchedKeySet(`{"index": 3}`)

		givenLastSeenRevisionForcedTo(firstEventRevision - 1)

		calledCh := make(chan struct{}, 2)
		givenEventHandler(func() {
			// simulate other node already made progress
			givenLastSeenRevisionForcedTo(firstEventRevision + 1)
			calledCh <- struct{}{}
		})

		whenPathWatcherStarted(testCtx)

		thenExactlyOneEventProcessed(calledCh)
	})

	It("Triggers a refresh using last seen value", func() {
		testCtx, waitForDone := synchronizeTest()
		defer waitForDone()

		firstEventRevision := givenWatchedKeySet(`{"index": 1}`)
		givenWatchedKeySet(`{"index": 2}`)

		givenLastSeenRevisionForcedTo(firstEventRevision - 1)

		thirdEventRevision := make(chan int64, 1)
		givenEventHandler(func() {
			givenLastSeenRevisionForcedTo(firstEventRevision + 1)
			thirdEventRevision <- givenWatchedKeySet(`{"index": 3}`)
		})

		whenPathWatcherStarted(testCtx)

		expectedRevision := <-thirdEventRevision + 1 //1 for update of the value
		thenWatchedKeyUpdatedTo(expectedRevision, `{"index": 3}`)
	})
})
