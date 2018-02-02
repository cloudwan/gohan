package goplugin

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("HTTP client test", func() {
	var (
		client *HTTP
		server *ghttp.Server
	)

	BeforeEach(func() {
		client = &HTTP{}
		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Reset()
	})

	It("Wait for response", func() {
		body := []byte(`abc`)
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/path"),
				ghttp.VerifyBody(body),
				ghttp.RespondWith(http.StatusOK, "hello"),
				ghttp.VerifyHeaderKV("hello", "world"),
			),
		)
		headers := map[string]string{"hello": "world"}
		resp, err := client.RequestRaw(context.Background(), http.MethodPost, server.URL()+"/path", headers, string(body))
		Expect(err).To(BeNil())
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body).To(Equal("hello"))
	})

	It("Handle cancel in context", func() {
		body := []byte(`abc`)
		headers := map[string]string{"hello": "world"}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.RequestRaw(ctx, http.MethodPost, server.URL()+"/path", headers, string(body))
		Expect(err).To(Equal(context.Canceled))
	})
})
