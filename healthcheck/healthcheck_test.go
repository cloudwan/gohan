package healthcheck

import (
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health check", func() {
	var config *util.Config
	BeforeEach(func() {
		config = util.NewConfig(map[string]interface{}{})
	})
	It("should return nil when Health Check is disabled", func() {
		config = util.NewConfig(map[string]interface{}{"healthcheck": map[string]interface{}{"enabled": false}})

		Expect(NewHealthCheck(nil, nil, "", config)).To(BeNil())
	})
	It("should be disabled by default", func() {
		Expect(NewHealthCheck(nil, nil, "", config)).To(BeNil())
	})
	It("should detect incorrect server address", func() {
		_, err := getHealthCheckAddress("localhost", config)

		Expect(err).To(MatchError("Incorrect gohan server address: localhost"))
	})
	It("should detect incorrect server port", func() {
		_, err := getHealthCheckAddress("localhost:test", config)

		Expect(err).To(MatchError("Incorrect gohan server address: expected port number got test"))
	})
	It("should not allow same address and port for server and Health Check", func() {
		config := util.NewConfig(map[string]interface{}{"healthcheck": map[string]interface{}{"address": ":1234"}})

		_, err := getHealthCheckAddress(":1234", config)

		Expect(err).To(MatchError("HealthCheck address must be different than server address :1234"))
	})
	It("should start Health Check on server port + 1 by default", func() {
		Expect(getHealthCheckAddress("localhost:1234", config)).To(Equal("localhost:1235"))
	})
})
