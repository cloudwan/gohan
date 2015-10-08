package pagination_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPagination(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pagination Suite")
}
