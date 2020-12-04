package schema

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tenancy", func() {
	var (
		tenantID = "some tenant id"
		domainID = "some domain id"
	)

	DescribeTable("Create tenancy",
		func(dataMap map[string]interface{}, expectedTenancy *Tenancy) {
			Expect(NewTenancy(dataMap)).To(Equal(expectedTenancy))
		},
		Entry("Should create tenancy with both tenant id and domain id",
			map[string]interface{}{
				tenantIDKey: tenantID,
				domainIDKey: domainID,
			},
			&Tenancy{
				TenantID: &tenantID,
				DomainID: &domainID,
			}),
		Entry("Should create tenancy with only tenant id",
			map[string]interface{}{
				tenantIDKey: tenantID,
			},
			&Tenancy{
				TenantID: &tenantID,
			}),
		Entry("Should create tenancy with only domain id",
			map[string]interface{}{
				domainIDKey: domainID,
			},
			&Tenancy{
				DomainID: &domainID,
			}),
		Entry("Should create tenancy with null tenant id",
			map[string]interface{}{
				tenantIDKey: nil,
			},
			&Tenancy{}),
	)
})
