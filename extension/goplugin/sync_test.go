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

package goplugin_test

import (
	"time"

	"context"

	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/sync/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sync", func() {
	var (
		mockCtrl *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("cancels Watch of context cancel", func() {
		mockSync := mock_sync.NewMockSync(mockCtrl)
		mockSync.EXPECT().WatchContext(gomock.Any(), "key", int64(1)).Return(make(chan *sync.Event, 1), nil)

		mockEnv := goplugin.Environment{}
		mockEnv.SetSync(mockSync)
		sut := mockEnv.Sync()

		doneCh := make(chan bool, 1)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			defer GinkgoRecover()
			sut.Watch(ctx, "key", time.Hour, 1)
			doneCh <- true
		}()

		cancel()
		Eventually(doneCh).Should(Receive())
	})
})
