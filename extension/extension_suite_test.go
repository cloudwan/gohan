package extension_test

import (
	"os"
	"testing"

	"github.com/cloudwan/gohan/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testDB1 db.DB
	testDB2 db.DB
)

func TestExtension(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extension Suite")
}

var _ = Describe("Suite set up and tear down", func() {
	const (
		testDBFile1 = "./extensionTest1.db"
		testDBFile2 = "./extensionTest2.db"
	)

	var _ = BeforeSuite(func() {
		var err error
		testDB1, err = db.ConnectDB("sqlite3", testDBFile1)
		Expect(err).NotTo(HaveOccurred())
		testDB2, err = db.ConnectDB("sqlite3", testDBFile2)
		Expect(err).NotTo(HaveOccurred())
	})

	var _ = AfterSuite(func() {
		os.Remove(testDBFile1)
		os.Remove(testDBFile2)
	})
})
