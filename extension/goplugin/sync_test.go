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
	"context"
	"time"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/sync/etcdv3"
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
		mockSync.EXPECT().WatchContext(gomock.Any(), "key", int64(1)).Return(make(chan *sync.Event, 1))

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

	It("returns nil data on delete", func() {
		rawSync, err := etcdv3.NewSync([]string{"localhost:2379"}, time.Second)
		Expect(err).NotTo(HaveOccurred())

		ctx := context.Background()

		const testKey = "/test/goplugin/sync"
		Expect(rawSync.Update(ctx, testKey, "{}")).To(Succeed())

		node, err := rawSync.Fetch(ctx, testKey)
		Expect(err).NotTo(HaveOccurred())

		env := goplugin.Environment{}
		env.SetSync(rawSync)
		gopluginSync := env.Sync()

		watchDone := make(chan struct{})

		var events []*goext.Event
		go func() {
			events, err = gopluginSync.Watch(context.Background(), testKey, time.Second*5, node.Revision+1)
			watchDone <- struct{}{}
		}()

		// assuming the Watch will start in (up to) 500ms
		<-time.After(time.Millisecond * 500)
		Expect(rawSync.Delete(ctx, testKey, false)).To(Succeed())

		Eventually(watchDone).Should(Receive())

		Expect(events).To(HaveLen(1))
		Expect(events[0].Action).To(Equal("delete"))
		Expect(events[0].Data).To(BeNil())
	})
})
