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
	l "github.com/cloudwan/gohan/log"
	srv "github.com/cloudwan/gohan/server"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sync watcher test", func() {
	const watchedKey = "/path_watcher/test"

	var (
		ctrl *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		//Expect(server.GetSync().Delete(
		//	context.Background(),
		//	srv.SyncWatchRevisionPrefix+watchedKey,
		//	false),
		//).To(Succeed())
	})

	AfterEach(func() {
		ctrl.Finish()
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

	shouldReceiveExactlyOnce := func(log l.Logger, ch <-chan struct{}) {
		log.Error("1")
		Eventually(ch).Should(Receive(), "the channel did not receive anything")
		Consistently(ch, time.Second).ShouldNot(Receive(), "the channel received twice")
		log.Error("2")
	}

	FIt("Stops processing when conflict detected", func() {
		log := l.NewLogger()
		log.Error("start")
		wg := sync.WaitGroup{}
		defer wg.Wait()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		Expect(server.GetSync().Delete(ctx, watchedKey, true)).To(Succeed())
		Expect(server.GetSync().Update(ctx, watchedKey, `{"index": 1}`)).To(Succeed())

		node, err := server.GetSync().Fetch(ctx, watchedKey)
		Expect(err).NotTo(HaveOccurred())
		firstEventRevision := node.Revision
		log.Info("first: %d", firstEventRevision)

		Expect(server.GetSync().Update(ctx, watchedKey, `{"index": 2}`)).To(Succeed())
		node, err = server.GetSync().Fetch(ctx, watchedKey)
		Expect(err).NotTo(HaveOccurred())
		log.Info("second: %d", node.Revision)

		Expect(server.GetSync().Update(ctx, watchedKey, `{"index": 3}`)).To(Succeed())
		node, err = server.GetSync().Fetch(ctx, watchedKey)
		Expect(err).NotTo(HaveOccurred())
		log.Info("third: %d", node.Revision)

		Expect(server.GetSync().Update(ctx, srv.SyncWatchRevisionPrefix+watchedKey, strconv.FormatInt(firstEventRevision-1, 10))).To(Succeed())

		calledCh := make(chan struct{}, 2)

		mockEnv := mock_extension.NewMockEnvironment(ctrl)
		mockEnv.EXPECT().Clone().Return(mockEnv).Times(2)
		mockEnv.EXPECT().HandleEvent("notification", gomock.Any()).DoAndReturn(func(interface{}, interface{}) error {
			log.Notice("executing ext 1")
			// simulate other node already made progress
			Expect(server.GetSync().Update(ctx, srv.SyncWatchRevisionPrefix+watchedKey, strconv.FormatInt(firstEventRevision+1, 10))).To(Succeed())
			log.Notice("executing ext 2")
			calledCh <- struct{}{}
			log.Notice("executing ext 3")
			return nil
		})

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

		shouldReceiveExactlyOnce(log, calledCh)

		log.Error("done")
	})
})
