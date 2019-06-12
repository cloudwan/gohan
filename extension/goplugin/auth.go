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

package goplugin

import (
	"fmt"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
)

// Auth is an implementation of IAuth
type Auth struct{}

func (a *Auth) HasRole(context goext.Context, principal string) bool {
	role, err := getRoleFromContext(context)
	if err != nil {
		log.Warning("HasRole: %s", err.Error())
		return false
	}

	return role.Match(principal)
}

func (a *Auth) GetTenantName(context goext.Context) string {
	auth, err := getAuthFromContext(context)
	if err != nil {
		log.Warning("GetTenantName: %s", err.Error())
		return ""
	}

	return auth.TenantName()
}

func (a *Auth) GetTenantID(context goext.Context) string {
	auth, err := getAuthFromContext(context)
	if err != nil {
		log.Warning("GetTenantID: %s", err.Error())
		return ""
	}

	return auth.TenantID()
}

func (a *Auth) GetDomainID(context goext.Context) string {
	auth, err := getAuthFromContext(context)
	if err != nil {
		log.Warning("GetDomainID: %s", err.Error())
		return ""
	}

	return auth.DomainID()
}

// IsAdmin returns true if user had admin role
func (a *Auth) IsAdmin(context goext.Context) bool {
	auth, err := getAuthFromContext(context)
	if err != nil {
		log.Warning("IsAdmin: %s", err.Error())
		return false
	}

	return auth.IsAdmin()
}

func (a *Auth) ValidateTenantID(ctx goext.Context, id string) (bool, error) {
	identityService, err := getIdentityServiceFromContext(ctx)
	if err != nil {
		return false, err
	}
	return identityService.ValidateTenantID(id)
}

func (a *Auth) ValidateDomainID(ctx goext.Context, id string) (bool, error) {
	identityService, err := getIdentityServiceFromContext(ctx)
	if err != nil {
		return false, err
	}
	return identityService.ValidateDomainID(id)
}

func getIdentityServiceFromContext(ctx goext.Context) (middleware.IdentityService, error) {
	rawIdentityService, err := getFromContext(ctx, "identity_service")
	if err != nil {
		return nil, err
	}

	identityService, ok := rawIdentityService.(middleware.IdentityService)
	if !ok {
		return nil, logWarnAndReturnErr(newInvalidTypeErr("identity_service"))
	}
	return identityService, nil
}

func getFromContext(ctx goext.Context, key string) (interface{}, error) {
	raw, ok := ctx[key]
	if !ok {
		return nil, logWarnAndReturnErr(newMissingInContextErr(key))
	}
	return raw, nil
}

func getRoleFromContext(context goext.Context) (*schema.Role, error) {
	roleRaw, err := getFromContext(context, "role")
	if err != nil {
		return nil, err
	}

	role, ok := roleRaw.(*schema.Role)
	if !ok {
		return nil, logWarnAndReturnErr(newInvalidTypeErr("role"))
	}

	return role, nil
}

func getAuthFromContext(context goext.Context) (schema.Authorization, error) {
	authRaw, err := getFromContext(context, "auth")
	if err != nil {
		return nil, err
	}

	auth, ok := authRaw.(schema.Authorization)
	if !ok {
		return nil, logWarnAndReturnErr(newInvalidTypeErr("auth"))
	}

	return auth, nil
}

func logWarnAndReturnErr(err error) error {
	log.Warning("%s", err)
	return err
}

func newMissingInContextErr(field string) error {
	return fmt.Errorf("missing '%s' field in context", field)
}

func newInvalidTypeErr(field string) error {
	return fmt.Errorf("invalid type of '%s' field in context", field)
}
