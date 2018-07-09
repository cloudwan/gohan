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
	"fmt"
	"time"

	"github.com/cloudwan/gohan/db/mocks"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/db/transaction/mocks"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database", func() {
	var (
		db       *goplugin.Database
		mockCtrl *gomock.Controller
		mockDB   *mock_db.MockDB
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDB = mock_db.NewMockDB(mockCtrl)
		mockDB.EXPECT().Options().Return(options.Default())
		db = goplugin.NewDatabase(mockDB)
	})

	Describe("Error handling", func() {
		Context("Begin", func() {
			It("should return an error on error", func() {
				expectedErr := fmt.Errorf("dummy error")
				mockDB.EXPECT().Begin().Return(&mocks.MockTransaction{}, expectedErr)

				tx, err := db.Begin()

				Expect(err).To(Equal(expectedErr))
				Expect(tx).To(BeNil())
			})

			It("should return an error on nil transaction received", func() {
				mockDB.EXPECT().Begin().Return(nil, nil)

				tx, err := db.Begin()

				Expect(err).To(HaveOccurred())
				Expect(tx).To(BeNil())
			})
		})

		Context("BeginTx", func() {
			It("should return an error on error", func() {
				expectedErr := fmt.Errorf("dummy error")
				mockDB.EXPECT().Begin(gomock.Any(), gomock.Any()).Return(&mocks.MockTransaction{}, expectedErr)

				tx, err := db.BeginTx(goext.MakeContext(), &goext.TxOptions{})

				Expect(err).To(Equal(expectedErr))
				Expect(tx).To(BeNil())
			})

			It("should return an error on nil transaction received", func() {
				mockDB.EXPECT().Begin(gomock.Any(), gomock.Any()).Return(nil, nil)

				tx, err := db.BeginTx(goext.MakeContext(), &goext.TxOptions{})

				Expect(err).To(HaveOccurred())
				Expect(tx).To(BeNil())
			})

			It("should call inner function using Context with Timeout", func() {
				tempCtxWithTimeout, _ := context.WithTimeout(context.Background(), 50*time.Millisecond)
				ctx := goext.Context{
					"context": tempCtxWithTimeout,
				}
				mockDB.EXPECT().Begin(gomock.Any(), gomock.Any()).Return(nil, nil).
					Do(func(options ...transaction.Option) {
						txParams := transaction.NewTxParams(options...)
						_, hasDeadline := txParams.Context.Deadline()
						Expect(hasDeadline).To(BeTrue())
					})

				db.BeginTx(ctx, &goext.TxOptions{})
			})
		})

		Context("Panic in transaction", func() {
			It("Should defer rollback", func() {
				type MockTX struct {
					*goext.MockITransaction
					closed bool
				}

				tx := &MockTX{
					MockITransaction: goext.NewMockITransaction(mockCtrl),
					closed:           false,
				}
				tx.EXPECT().Closed().Return(tx.closed)
				tx.EXPECT().Close().DoAndReturn(func() error {
					tx.closed = true
					return nil
				})

				context := goext.Context{}

				env := goplugin.NewMockIEnvironment(nil, nil)
				env.SetMockModules(goext.MockModules{
					Util:            true,
					Database:        true,
					DefaultDatabase: true,
				})
				env.MockUtil().EXPECT().GetTransaction(gomock.Any()).Return(
					nil, false,
				)
				env.MockDatabase().EXPECT().Options().Return(
					goext.DbOptions{RetryTxCount: 1},
				)
				env.MockDatabase().EXPECT().Begin().Return(tx, nil)

				Expect(func() {
					env.Database().Within(context, func(tx goext.ITransaction) error {
						Expect(context["transaction"]).To(Equal(tx))
						panic("test")
					})
				}).To(Panic())
				Expect(context).ToNot(HaveKey("transaction"))
				Expect(tx.closed).To(BeTrue())
			})
		})
	})

})
