package log_test

import (
	"io/ioutil"
	"os"

	"github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Log", func() {
	Context("Tracing logger", func() {
		var (
			mockStdErrWriter *os.File
			mockStdErrReader *os.File
			realStderr       *os.File
		)

		BeforeEach(func() {
			realStderr = os.Stderr
			mockStdErrReader, mockStdErrWriter, _ = os.Pipe()
			os.Stderr = mockStdErrWriter

			config := util.NewConfig(
				map[string]interface{}{
					"logging/stderr/enabled": true,
				},
			)
			Expect(log.SetUpLogging(config)).To(Succeed())
		})

		AfterEach(func() {
			os.Stderr = realStderr
		})

		It("outputs traceId", func() {
			logger := log.NewLogger(log.TraceId("test-trace-id"))

			logger.Info("log message")
			mockStdErrWriter.Close()

			stderr, err := ioutil.ReadAll(mockStdErrReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(stderr)).To(ContainSubstring("test-trace-id"))
		})
	})
})
