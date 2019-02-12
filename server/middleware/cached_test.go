package middleware

import (
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/schema"
)

var _ = ginkgo.Describe("Cached identity service", func() {

	var (
		cachedIdentityService IdentityService
		mockedIdentityService *MockIdentityService
		auth                  schema.Authorization
		tenantID              string
		tenantName            string
		token                 string
		ctrl                  *gomock.Controller
	)

	ginkgo.BeforeEach(func() {
		ctrl = gomock.NewController(ginkgo.GinkgoT())
		mockedIdentityService = NewMockIdentityService(ctrl)
		cachedIdentityService = NewCachedIdentityService(mockedIdentityService, time.Second)
		tenant := schema.Tenant{
			ID:   "tenant-id",
			Name: "tenant-name",
		}
		token = "token"
		auth = schema.NewAuthorizationBuilder().
			WithTenant(tenant).
			BuildScopedToTenant()
	})

	ginkgo.AfterEach(func() {
		ctrl.Finish()
	})

	ginkgo.It("Use inner service if authorization is not cached and save returned one in cache", func() {
		mockedIdentityService.EXPECT().VerifyToken(token).Return(auth, nil).Times(1)
		rv, err := cachedIdentityService.VerifyToken(token)
		Expect(rv).To(Equal(auth))
		Expect(err).To(BeNil())
		rv, err = cachedIdentityService.VerifyToken(token)
		Expect(rv).To(Equal(auth))
		Expect(err).To(BeNil())
	})

	ginkgo.It("Pass GetTenantID to inner service", func() {
		mockedIdentityService.EXPECT().GetTenantID(tenantName).Return(tenantID, nil)
		rv, err := cachedIdentityService.GetTenantID(tenantName)
		Expect(rv).To(Equal(tenantID))
		Expect(err).To(BeNil())

	})

	ginkgo.It("Cache tenant names", func() {
		mockedIdentityService.EXPECT().GetTenantName(tenantID).Return(tenantName, nil).Times(1)
		rv, err := cachedIdentityService.GetTenantName(tenantID)
		Expect(rv).To(Equal(tenantName))
		Expect(err).To(BeNil())
		rv, err = cachedIdentityService.GetTenantName(tenantID)
		Expect(rv).To(Equal(tenantName))
		Expect(err).To(BeNil())
	})

	ginkgo.It("Uses client Token during GetServiceAuthorization", func() {
		mockedIdentityService.EXPECT().GetServiceTokenID().Return(token)
		mockedIdentityService.EXPECT().VerifyToken(token).Return(auth, nil)
		rv, err := cachedIdentityService.GetServiceAuthorization()
		Expect(rv).To(Equal(auth))
		Expect(err).To(BeNil())
	})
})
