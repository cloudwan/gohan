// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package cloud

func getV2TokensResponse() interface{} {
	return map[string]interface{}{
		"access": map[string]interface{}{
			"token": map[string]interface{}{
				"expires":   "2014-01-31T15:30:58Z",
				"id":        "admin_token",
				"issued_at": "2014-01-30T15:30:58.819584",
				"tenant": map[string]interface{}{
					"description": nil,
					"enabled":     true,
					"id":          "1234",
					"name":        "admin",
				},
			},
		},
	}
}

func getV3TokensResponse() interface{} {
	return map[string]interface{}{
		"token": map[string]interface{}{
			"expires_at": "2013-02-27T18:30:59.999999Z",
			"issued_at":  "2013-02-27T16:30:59.999999Z",
			"methods": []string{
				"password",
			},
			"user": map[string]interface{}{
				"domain": map[string]interface{}{
					"id":   "111",
					"name": "domain",
				},
				"id":   "1234",
				"name": "admin",
			},
		},
	}
}

func getV2TenantsResponse() interface{} {
	return map[string]interface{}{
		"tenants": []interface{}{
			map[string]interface{}{
				"id":          "1234",
				"name":        "admin",
				"description": "admin description",
				"enabled":     true,
			},
			map[string]interface{}{
				"id":          "3456",
				"name":        "demo",
				"description": "demo description",
				"enabled":     true,
			},
		},
	}
}

func getV3TenantsResponse() interface{} {
	return map[string]interface{}{
		"projects": []interface{}{
			map[string]interface{}{
				"id":          "1234",
				"name":        "admin",
				"description": "admin description",
				"enabled":     true,
			},
			map[string]interface{}{
				"id":          "3456",
				"name":        "demo",
				"description": "demo description",
				"enabled":     true,
			},
		},
	}
}
