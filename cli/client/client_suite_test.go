package client_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cli Suite")
}

var _ = Describe("Suit set up and tear down", func() {
	var _ = BeforeSuite(func() {
		os.Unsetenv("OS_AUTH_URL")
		os.Unsetenv("OS_USERNAME")
		os.Unsetenv("OS_USERID")
		os.Unsetenv("OS_PASSWORD")
		os.Unsetenv("OS_TENANT_ID")
		os.Unsetenv("OS_TENANT_NAME")
		os.Unsetenv("OS_DOMAIN_ID")
		os.Unsetenv("OS_DOMAIN_NAME")
	})
})
