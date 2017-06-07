package transaction_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var ctrl *gomock.Controller

func TestTransaction(t *testing.T) {
	RegisterFailHandler(Fail)
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	RunSpecs(t, "Transaction Suite")
}
