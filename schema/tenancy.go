package schema

const (
	tenantIDKey = "tenant_id"
	domainIDKey = "domain_id"
)

type Tenancy struct {
	TenantID *string
	DomainID *string
}

func NewTenancy(data map[string]interface{}) *Tenancy {
	return &Tenancy{
		TenantID: interfaceToStringPointer(data[tenantIDKey]),
		DomainID: interfaceToStringPointer(data[domainIDKey]),
	}
}

func interfaceToStringPointer(s interface{}) *string {
	value, ok := s.(string)
	if !ok {
		return nil
	}
	return &value
}
