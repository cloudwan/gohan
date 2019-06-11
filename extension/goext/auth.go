// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goext

// IAuth is an interface to auth in Gohan
type IAuth interface {
	// HasRole reports whether context has given role
	HasRole(context Context, role string) bool
	// GetTenantName return name from the given context
	GetTenantName(context Context) string
	// IsAdmin reports whether context belongs to admin
	IsAdmin(context Context) bool
	// ValidateTenantID checks whether given tenant_id exists in backend.
	ValidateTenantID(ctx Context, id string) (bool, error)
	// ValidateDomainID checks whether given domain_id exists in backend.
	ValidateDomainID(ctx Context, id string) (bool, error)
	// ValidateTenantIDAndDomainIDPair checks whether given tenant_id is a child of domain with domain_id.
	ValidateTenantIDAndDomainIDPair(ctx Context, tenantID, domainID string) (bool, error)
}
