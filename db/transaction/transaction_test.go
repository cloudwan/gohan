package transaction_test

import (
	"errors"

	"time"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/db/transaction/mocks"
	"github.com/cloudwan/gohan/schema"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retryable Transaction", func() {

	var any = gomock.Any()
	var retryConfig transaction.RetryConfig

	var givenNumberOfAttempts = func(attempts int) {
		retryConfig.Attempts = attempts
	}

	var givenSleepInterval = func(interval time.Duration) {
		retryConfig.SleepInterval = interval
	}

	var givenRetryStrategy = func(strategy transaction.RetryStrategyPredicate) {
		retryConfig.Strategy = strategy
	}

	BeforeEach(func() {
		givenRetryStrategy(transaction.IsDeadlock)
		givenNumberOfAttempts(2)
		givenSleepInterval(0 * time.Millisecond)
	})

	Context("Retry logic", func() {
		var rawTxMock *mocks.MockTransaction
		var sut transaction.RetryableTransaction

		BeforeEach(func() {
			rawTxMock = mocks.NewMockTransaction(ctrl)
			givenNumberOfAttempts(2)
			sut = transaction.RetryableTransaction{
				Tx:     rawTxMock,
				Config: retryConfig,
			}
		})

		It("Number of attempts specified by config", func() {
			givenNumberOfAttempts(3)
			sut = transaction.RetryableTransaction{
				Tx:     rawTxMock,
				Config: retryConfig,
			}
			err := errors.New(transaction.MYSQL_DEADLOCK_MSG)
			rawTxMock.EXPECT().Create(any).Return(err)
			rawTxMock.EXPECT().Create(any).Return(err)
			rawTxMock.EXPECT().Create(any).Return(err)
			Expect(sut.Create(nil)).To(Equal(err))
		})

		It("No retry if error is not recognized by strategy", func() {
			err := errors.New("Other error")
			rawTxMock.EXPECT().Create(any).Return(err)
			Expect(sut.Create(nil)).To(Equal(err))
		})

		It("No retry if error is nil", func() {
			rawTxMock.EXPECT().Create(any).Return(nil)
			Expect(sut.Create(nil)).To(Succeed())
		})
	})

	Context("Mysql backend", func() {
		var rawTxMock *mocks.MockTransaction
		var sut transaction.RetryableTransaction

		BeforeEach(func() {
			rawTxMock = mocks.NewMockTransaction(ctrl)
			sut = transaction.RetryableTransaction{
				Tx:     rawTxMock,
				Config: retryConfig,
			}
		})

		It("Retry Create on deadlock", func() {
			rawTxMock.EXPECT().Create(any).Return(errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().Create(any).Return(nil)
			Expect(sut.Create(nil)).To(Succeed())
		})

		It("Retry Update on deadlock", func() {
			rawTxMock.EXPECT().Update(any).Return(errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().Update(any).Return(nil)
			Expect(sut.Update(nil)).To(Succeed())
		})
		It("Retry StateUpdate on deadlock", func() {
			rawTxMock.EXPECT().StateUpdate(any, any).Return(errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().StateUpdate(any, any).Return(nil)
			Expect(sut.StateUpdate(nil, nil)).To(Succeed())
		})
		It("Retry Delete on deadlock", func() {
			rawTxMock.EXPECT().Delete(any, any).Return(errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().Delete(any, any).Return(nil)
			Expect(sut.Delete(nil, nil)).To(Succeed())
		})
		It("Retry Fetch on deadlock", func() {
			rawTxMock.EXPECT().Fetch(any, any).Return(nil, errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().Fetch(any, any).Return(nil, nil)
			_, err := sut.Fetch(nil, nil)
			Expect(err).To(BeNil())
		})
		It("Retry LockFetch on deadlock", func() {
			rawTxMock.EXPECT().LockFetch(any, any, any).Return(nil, errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().LockFetch(any, any, any).Return(nil, nil)
			_, err := sut.LockFetch(nil, nil, schema.NoLocking)
			Expect(err).To(BeNil())
		})
		It("Retry StateFetch on deadlock", func() {
			rv := transaction.ResourceState{}
			rawTxMock.EXPECT().StateFetch(any, any).Return(rv, errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().StateFetch(any, any).Return(rv, nil)
			_, err := sut.StateFetch(nil, nil)
			Expect(err).To(BeNil())
		})
		It("Retry List on deadlock", func() {
			rawTxMock.EXPECT().List(any, any, any, any).Return(nil, uint64(0), errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().List(any, any, any, any).Return(nil, uint64(0), nil)
			_, _, err := sut.List(nil, nil, nil, nil)
			Expect(err).To(BeNil())
		})
		It("Retry LockList on deadlock", func() {
			rawTxMock.EXPECT().LockList(any, any, any, any, any).Return(nil, uint64(0), errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().LockList(any, any, any, any, any).Return(nil, uint64(0), nil)
			_, _, err := sut.LockList(nil, nil, nil, nil, schema.NoLocking)
			Expect(err).To(BeNil())
		})
		It("Retry Query on deadlock", func() {
			rawTxMock.EXPECT().Query(any, any, any).Return(nil, errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().Query(any, any, any).Return(nil, nil)
			_, err := sut.Query(nil, "", nil)
			Expect(err).To(BeNil())
		})
		It("Retry Exec on deadlock", func() {
			rawTxMock.EXPECT().Exec(any, any).Return(errors.New(transaction.MYSQL_DEADLOCK_MSG))
			rawTxMock.EXPECT().Exec(any, any).Return(nil)
			Expect(sut.Exec("", nil)).To(Succeed())
		})

		It("Commit not retryable", func() {
			err := errors.New(transaction.MYSQL_DEADLOCK_MSG)
			rawTxMock.EXPECT().Commit().Return(err)
			Expect(sut.Commit()).To(Equal(err))
		})

		It("Close not retryable", func() {
			err := errors.New(transaction.MYSQL_DEADLOCK_MSG)
			rawTxMock.EXPECT().Close().Return(err)
			Expect(sut.Close()).To(Equal(err))
		})
	})

	Context("SQLite backend", func() {
		var rawTxMock *mocks.MockTransaction
		var sut transaction.RetryableTransaction

		BeforeEach(func() {
			rawTxMock = mocks.NewMockTransaction(ctrl)
			sut = transaction.RetryableTransaction{
				Tx:     rawTxMock,
				Config: retryConfig,
			}
		})

		It("Retry Create on deadlock", func() {
			rawTxMock.EXPECT().Create(any).Return(errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().Create(any).Return(nil)
			Expect(sut.Create(nil)).To(Succeed())
		})

		It("Retry Update on deadlock", func() {
			rawTxMock.EXPECT().Update(any).Return(errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().Update(any).Return(nil)
			Expect(sut.Update(nil)).To(Succeed())
		})
		It("Retry StateUpdate on deadlock", func() {
			rawTxMock.EXPECT().StateUpdate(any, any).Return(errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().StateUpdate(any, any).Return(nil)
			Expect(sut.StateUpdate(nil, nil)).To(Succeed())
		})
		It("Retry Delete on deadlock", func() {
			rawTxMock.EXPECT().Delete(any, any).Return(errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().Delete(any, any).Return(nil)
			Expect(sut.Delete(nil, nil)).To(Succeed())
		})
		It("Retry Fetch on deadlock", func() {
			rawTxMock.EXPECT().Fetch(any, any).Return(nil, errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().Fetch(any, any).Return(nil, nil)
			_, err := sut.Fetch(nil, nil)
			Expect(err).To(BeNil())
		})
		It("Retry LockFetch on deadlock", func() {
			rawTxMock.EXPECT().LockFetch(any, any, any).Return(nil, errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().LockFetch(any, any, any).Return(nil, nil)
			_, err := sut.LockFetch(nil, nil, schema.NoLocking)
			Expect(err).To(BeNil())
		})
		It("Retry StateFetch on deadlock", func() {
			rv := transaction.ResourceState{}
			rawTxMock.EXPECT().StateFetch(any, any).Return(rv, errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().StateFetch(any, any).Return(rv, nil)
			_, err := sut.StateFetch(nil, nil)
			Expect(err).To(BeNil())
		})
		It("Retry List on deadlock", func() {
			rawTxMock.EXPECT().List(any, any, any, any).Return(nil, uint64(0), errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().List(any, any, any, any).Return(nil, uint64(0), nil)
			_, _, err := sut.List(nil, nil, nil, nil)
			Expect(err).To(BeNil())
		})
		It("Retry LockList on deadlock", func() {
			rawTxMock.EXPECT().LockList(any, any, any, any, any).Return(nil, uint64(0), errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().LockList(any, any, any, any, any).Return(nil, uint64(0), nil)
			_, _, err := sut.LockList(nil, nil, nil, nil, schema.NoLocking)
			Expect(err).To(BeNil())
		})
		It("Retry Query on deadlock", func() {
			rawTxMock.EXPECT().Query(any, any, any).Return(nil, errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().Query(any, any, any).Return(nil, nil)
			_, err := sut.Query(nil, "", nil)
			Expect(err).To(BeNil())
		})
		It("Retry Exec on deadlock", func() {
			rawTxMock.EXPECT().Exec(any, any).Return(errors.New(transaction.SQLITE_DEADLOCK_MSG))
			rawTxMock.EXPECT().Exec(any, any).Return(nil)
			Expect(sut.Exec("", nil)).To(Succeed())
		})

		It("Commit not retryable", func() {
			err := errors.New(transaction.SQLITE_DEADLOCK_MSG)
			rawTxMock.EXPECT().Commit().Return(err)
			Expect(sut.Commit()).To(Equal(err))
		})

		It("Close not retryable", func() {
			err := errors.New(transaction.SQLITE_DEADLOCK_MSG)
			rawTxMock.EXPECT().Close().Return(err)
			Expect(sut.Close()).To(Equal(err))
		})
	})
})
