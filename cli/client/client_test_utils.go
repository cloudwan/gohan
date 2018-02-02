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

package client

import (
	"encoding/json"
	"sort"

	"github.com/cloudwan/gohan/schema"
	g "github.com/onsi/gomega"
)

func getAuthRequest() interface{} {
	return map[string]interface{}{
		"auth": map[string]interface{}{
			"tenantName": "admin",
			"passwordCredentials": map[string]interface{}{
				"username": "admin",
				"password": "password",
			},
		},
	}
}

func getAuthResponse(gohanEndpointURL string) interface{} {
	return map[string]interface{}{
		"access": map[string]interface{}{
			"token": map[string]interface{}{
				"expires":   "2014-01-31T15:30:58Z",
				"id":        "admin_token",
				"issued_at": "2014-01-30T15:30:58.819584",
				"tenant": map[string]interface{}{
					"description": nil,
					"enabled":     true,
					"id":          "fc394f2ab2df4114bde39905f800dc57",
					"name":        "admin",
				},
			},
			"serviceCatalog": []map[string]interface{}{
				{
					"endpoints": []map[string]interface{}{
						{
							"adminURL":    gohanEndpointURL,
							"internalURL": gohanEndpointURL,
							"publicURL":   gohanEndpointURL,
							"region":      "RegionOne",
							"id":          "2dad48f09e2a447a9bf852bcd93548ef",
						},
					},
					"endpoints_links": []interface{}{},
					"type":            "gohan",
					"name":            "gohan",
				},
			},
		},
	}
}

func getCastleSchema() map[string]interface{} {
	return map[string]interface{}{
		"description": "Castle",
		"id":          "castle",
		"singular":    "castle",
		"title":       "castle",
		"parent":      "",
		"plural":      "castles",
		"prefix":      "/v2.0",
		"url":         "/v2.0/castles",
		"schema": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"format": "uuid",
					"permission": []interface{}{
						"create",
					},
					"type": "string",
				},
				"name": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "string",
				},
			},
			"propertiesOrder": []interface{}{
				"id",
				"name",
			},
		},
		"actions": map[string]interface{}{
			"open_gates": map[string]interface{}{
				"method": "GET",
				"path":   "/:id/open_gates",
				"input": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"gate_id": map[string]interface{}{
							"type": "number",
						},
					},
				},
				"output": map[string]interface{}{
					"type": "string",
				},
			},
			"close_gates": map[string]interface{}{
				"method": "GET",
				"path":   "/close_all_gates",
				"output": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}
}

func getTowerSchema() map[string]interface{} {
	return map[string]interface{}{
		"description": "Tower",
		"id":          "tower",
		"title":       "tower",
		"singular":    "tower",
		"parent":      "castle",
		"plural":      "towers",
		"prefix":      "/v2.0",
		"url":         "/v2.0/towers",
		"schema": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"format": "uuid",
					"permission": []interface{}{
						"create",
					},
					"type": "string",
				},
				"isMain": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "boolean",
				},
				"sister_id": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type":     "string",
					"relation": "tower",
				},
			},
			"propertiesOrder": []interface{}{
				"id",
				"isMain",
				"sister",
			},
		},
	}
}

func getChamberSchema() map[string]interface{} {
	return map[string]interface{}{
		"description": "Chamber",
		"id":          "chamber",
		"title":       "chamber",
		"singular":    "chamber",
		"parent":      "tower",
		"plural":      "chambers",
		"prefix":      "/v2.0",
		"url":         "/v2.0/chambers",
		"schema": map[string]interface{}{
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"permission": []interface{}{
						"create",
					},
					"type": []string{"string", "null"},
				},
				"isPrincessIn": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "boolean",
				},
				"windows": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "integer",
				},
				"chest": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"weapon": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			"propertiesOrder": []interface{}{
				"name",
				"isPrincessIn",
				"windows",
			},
		},
	}
}

func getActionsSchema() map[string]interface{} {
	return map[string]interface{}{
		"description": "Action",
		"id":          "action",
		"title":       "action",
		"singular":    "action",
		"plural":      "action",
		"prefix":      "/v2.0",
		"url":         "/v2.0/actions",
		"schema": map[string]interface{}{
			"properties": map[string]interface{}{
				"a": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "number",
				},
				"b": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "boolean",
				},
				"c": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "string",
				},
				"d": map[string]interface{}{
					"permission": []interface{}{
						"create",
						"update",
					},
					"type": "object",
				},
			},
			"propertiesOrder": []interface{}{
				"a",
				"b",
				"c",
				"d",
			},
		},
		"actions": map[string]interface{}{
			"do_a": map[string]interface{}{
				"method": "GET",
				"path":   "/do_a",
				"output": map[string]interface{}{
					"type": "string",
				},
			},
			"do_b": map[string]interface{}{
				"method": "GET",
				"path":   "/:id/do_b",
			},
			"do_c": map[string]interface{}{
				"method": "GET",
				"path":   "/do_c",
				"input": map[string]interface{}{
					"type": "number",
				},
			},
			"do_d": map[string]interface{}{
				"method": "GET",
				"path":   "/:id/do_d",
				"input": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a_in": map[string]interface{}{
							"type": "number",
						},
						"b_in": map[string]interface{}{
							"type": "string",
						},
						"c_in": map[string]interface{}{
							"type": "boolean",
						},
					},
				},
			},
		},
	}
}

func getSchemasResponse() interface{} {
	return map[string]interface{}{
		"schemas": []map[string]interface{}{
			getCastleSchema(),
			getTowerSchema(),
		},
	}
}

func getActionSchemasResponse() interface{} {
	return map[string]interface{}{
		"schemas": []map[string]interface{}{
			getActionsSchema(),
		},
	}
}

func getSchemas() []*schema.Schema {
	rawSchemasResponse := getSchemasResponse()
	rawSchemas := rawSchemasResponse.(map[string]interface{})["schemas"].([]map[string]interface{})
	schemas := []*schema.Schema{}
	for _, s := range rawSchemas {
		schema, _ := schema.NewSchemaFromObj(s)
		schemas = append(schemas, schema)
	}
	return schemas
}

var (
	castleID       = "de305d54-75b4-431b-adb2-eb6b9e546013"
	icyTowerID     = "de305d54-75b4-431b-adb2-eb6b9e546014"
	icyTowerName   = "Icy Tower"
	babylonTowerID = "de305d54-75b4-431b-adb2-eb6b9e546015"
)

func openGates() string {
	return "gates opened"
}

func closeGates() string {
	return "gates closed"
}

func doA() string {
	return "done a"
}

func getIcyTower() map[string]interface{} {
	return map[string]interface{}{
		"id":   icyTowerID,
		"name": icyTowerName,
	}
}

func getBabylonTower() map[string]interface{} {
	return map[string]interface{}{
		"id":   babylonTowerID,
		"name": "Babylon Tower",
	}
}

func getIcyTowerListResponse() map[string]interface{} {
	return map[string]interface{}{
		"towers": []map[string]interface{}{
			getIcyTower(),
		},
	}
}

func getTowerListResponse() map[string]interface{} {
	return map[string]interface{}{
		"towers": []map[string]interface{}{
			getIcyTower(),
			getBabylonTower(),
		},
	}
}

func getTowerListJSONResponse() []byte {
	towersJSON, _ := json.Marshal(getTowerListResponse())
	return towersJSON
}

func compareSchemas(actual, expected []*schema.Schema) {
	g.Expect(actual).To(g.HaveLen(len(expected)))
	for i, s := range actual {
		sortProperties(s)
		sortProperties(expected[i])
		schema.SortActions(s)
		schema.SortActions(expected[i])
		g.Expect(s).To(g.Equal(expected[i]))
	}
}

type properties []schema.Property

func (p properties) Len() int           { return len(p) }
func (p properties) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p properties) Less(i, j int) bool { return p[i].ID < p[j].ID }

func sortProperties(schema *schema.Schema) {
	sort.Sort(properties(schema.Properties))
}

//ByName is alias type for handling gohanCommands
type ByName []gohanCommand

func (a ByName) Len() int {
	return len(a)
}
func (a ByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }
