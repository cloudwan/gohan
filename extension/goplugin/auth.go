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
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/schema"
	"errors"
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

// IsAdmin returns true if user had admin role
func (a *Auth) IsAdmin(context goext.Context) bool {
	auth, err := getAuthFromContext(context)
	if err != nil {
		log.Warning("IsAdmin: %s", err.Error())
		return false
	}

	return auth.IsAdmin()
}

func getRoleFromContext(context goext.Context) (*schema.Role, error) {
	roleRaw, ok := context["role"]
	if !ok {
		log.Warning("missing 'role' field in context")
		return nil, errors.New("missing 'role' field in context")
	}

	role, ok := roleRaw.(*schema.Role)
	if !ok {
		log.Warning("invalid type of 'role' field in context")
		return nil, errors.New("invalid type of 'role' field in context")
	}

	return role, nil
}

func getAuthFromContext(context goext.Context) (schema.Authorization, error) {
	authRaw, ok := context["auth"]
	if !ok {
		return nil, errors.New("missing 'auth' field in context")
	}

	auth, ok := authRaw.(schema.Authorization)
	if !ok {
		return nil, errors.New("invalid type of 'auth' field in context")
	}

	return auth, nil
}
