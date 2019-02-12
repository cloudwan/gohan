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
	"sync"

	"github.com/cloudwan/gohan/extension"
	mock_extension "github.com/cloudwan/gohan/extension/mocks"
	srv "github.com/cloudwan/gohan/server"
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
})
