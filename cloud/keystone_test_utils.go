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
