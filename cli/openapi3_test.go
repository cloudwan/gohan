package cli

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/cloudwan/gohan/schema"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mohae/deepcopy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli"
)

var _ = Describe("OpenAPI v3", func() {
	const (
		openApiVersion = "3.0.0"

		configFile = "test/config.yaml"

		apiTitle       = "dummy-api-title"
		apiVersion     = "dummy-api-version"
		apiDescription = "dummy-api-description"
	)

	var (
		output []byte
		root   *openapi3.Swagger
	)

	BeforeEach(func() {
		output = command(
			getOpenAPI3Command(),
			"--config-file", configFile,
			"--title", apiTitle,
			"--version", apiVersion,
			"--description", apiDescription,
			"--template", "../etc/templates/openapi3.tmpl",
		)

		var err error
		root, err = openapi3.NewSwaggerLoader().LoadSwaggerFromData(output)
		Expect(err).NotTo(HaveOccurred())
		removeRef(root)
		schema.ClearManager()
	})

	It("Should be valid JSON", func() {
		// parser used by openapi3 package is too forgiving
		var m map[string]interface{}
		Expect(json.Unmarshal(output, &m)).To(Succeed())
	})

	It("Root should contain info section and OpenAPI version", func() {
		Expect(root.OpenAPI).To(Equal(openApiVersion))
		Expect(transformJSON(root.Info)).To(Equal(transformJSON(openapi3.Info{
			Title:       apiTitle,
			Version:     apiVersion,
			Description: apiDescription,
		})))
	})

	Describe("Properties", func() {
		It("can be nullable or deprecated", func() {
			expectedProperties := transformJSON(map[string]*openapi3.SchemaRef{
				"id": {Value: &openapi3.Schema{
					Description: "Apple ID description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Apple ID title",
					}},
				}},
				"color": {Value: &openapi3.Schema{
					Description: "Apple color description",
					Type:        "string",
					Nullable:    true,
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Apple color title",
					}},
				}},
				"taste": {Value: &openapi3.Schema{
					Description: "Apple taste description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title":      "Apple taste title",
						"deprecated": true,
					}},
				}},
			})

			properties := transformJSON(root.Components.Schemas["apple"].Value.Properties)
			Expect(properties).To(Equal(expectedProperties))
		})

		It("can have min, max, default, example values", func() {
			expectedProperties := transformJSON(map[string]*openapi3.SchemaRef{
				"sourness": {Value: &openapi3.Schema{
					Min:         number(0),
					Max:         number(100),
					Default:     number(50),
					Example:     number(42),
					Description: "Lemon sourness description",
					Type:        "integer",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Lemon sourness title",
					}},
				}},
			})

			properties := transformJSON(root.Components.Schemas["lemon"].Value.Properties)
			Expect(properties).To(Equal(expectedProperties))
		})

		It("can have additionalProperties", func() {
			expectedProperties := transformJSON(map[string]*openapi3.SchemaRef{
				"id": {Value: &openapi3.Schema{
					Description: "Orange ID description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Orange ID title",
					}},
				}},
				"content": {Value: &openapi3.Schema{
					Description: "Orange content description",
					Type:        "object",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Orange content title",
					}},
					AdditionalProperties: &openapi3.SchemaRef{Value: &openapi3.Schema{
						Description: "Orange content additional properties description",
						Type:        "object",
						ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
							"title": "Orange content additional properties title",
						}},
						Properties: map[string]*openapi3.SchemaRef{
							"foobar": {Value: &openapi3.Schema{
								Description: "Orange content foobar description",
								Type:        "string",
								ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
									"title": "Orange content foobar title",
								}},
							}},
						},
					}},
				}},
			})

			properties := transformJSON(root.Components.Schemas["orange"].Value.Properties)
			Expect(properties).To(Equal(expectedProperties))
		})
	})

	Describe("Related schemas", func() {
		responseContent := func(response *openapi3.ResponseRef) *openapi3.SchemaRef {
			return response.Value.Content.Get("application/json").Schema
		}
		requestContent := func(request *openapi3.RequestBodyRef) *openapi3.SchemaRef {
			return request.Value.Content.Get("application/json").Schema
		}
		withNoAdditionalPropertiesAllowed := func(schema *openapi3.SchemaRef) *openapi3.SchemaRef {
			schema = deepcopy.Copy(schema).(*openapi3.SchemaRef)
			schema.Value.AdditionalPropertiesAllowed = boolean(false)
			return schema
		}

		expectedColorSchema := &openapi3.SchemaRef{Value: &openapi3.Schema{
			Type: "object",
			Properties: map[string]*openapi3.SchemaRef{
				"id": {Value: &openapi3.Schema{
					Description: "Color ID description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Color ID title",
					}},
				}},
				"name": {Value: &openapi3.Schema{
					Description: "Color name description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Color name title",
					}},
				}},
			},
		}}

		expectedBananaSchemaWithRelation := &openapi3.SchemaRef{Value: &openapi3.Schema{
			Type: "object",
			Properties: map[string]*openapi3.SchemaRef{
				"id": {Value: &openapi3.Schema{
					Description: "Banana ID description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Banana ID title",
					}},
				}},
				"color_id": {Value: &openapi3.Schema{
					Description: "Banana color description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Banana color title",
					}},
				}},
				"color_object": expectedColorSchema,
			},
		}}

		expectedShopSchema := &openapi3.SchemaRef{Value: &openapi3.Schema{
			Type: "object",
			Properties: map[string]*openapi3.SchemaRef{
				"id": {Value: &openapi3.Schema{
					Description: "Shop ID description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Shop ID title",
					}},
				}},
				"banana_id": {Value: &openapi3.Schema{
					Description: "Banana description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Banana title",
					}},
				}},
			},
		}}

		expectedShopSchemaWithRelation := deepcopy.Copy(expectedShopSchema).(*openapi3.SchemaRef)
		expectedShopSchemaWithRelation.Value.Properties["banana_object"] = expectedBananaSchemaWithRelation

		It("GET single response should contain object with related schemas", func() {
			expectedSchema := transformJSON(expectedShopSchemaWithRelation)
			schema := transformJSON(responseContent(root.Paths["/shops/{id}"].Get.Responses["200"]))
			Expect(schema).To(Equal(expectedSchema))
		})

		It("GET multiple response should contain list of objects with related schemas", func() {
			expectedSchema := transformJSON(&openapi3.SchemaRef{Value: &openapi3.Schema{
				Type:  "array",
				Items: expectedShopSchemaWithRelation,
			}})
			response := transformJSON(responseContent(root.Paths["/shops"].Get.Responses["200"]))
			Expect(response).To(Equal(expectedSchema))
		})

		It("PUT request body should contain object without related schemas", func() {
			expectedSchema := transformJSON(withNoAdditionalPropertiesAllowed(expectedShopSchema))
			schema := transformJSON(requestContent(root.Paths["/shops/{id}"].Put.RequestBody))
			Expect(schema).To(Equal(expectedSchema))
		})

		It("PUT response should contain object without related schemas", func() {
			expectedSchema := transformJSON(expectedShopSchema)
			schema := transformJSON(responseContent(root.Paths["/shops/{id}"].Put.Responses["200"]))
			Expect(schema).To(Equal(expectedSchema))
		})

		It("POST request body should contain object without related schemas", func() {
			expectedSchema := transformJSON(withNoAdditionalPropertiesAllowed(expectedShopSchema))
			schema := transformJSON(requestContent(root.Paths["/shops"].Post.RequestBody))
			Expect(schema).To(Equal(expectedSchema))
		})

		It("POST response should contain object without related schemas", func() {
			expectedSchema := transformJSON(expectedShopSchema)
			schema := transformJSON(responseContent(root.Paths["/shops"].Post.Responses["201"]))
			Expect(schema).To(Equal(expectedSchema))
		})

		It("should not embed schemas from array properties with relation", func() {
			expectedSchema := transformJSON(&openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: "object",
				Properties: map[string]*openapi3.SchemaRef{
					"id": {Value: &openapi3.Schema{
						Description: "Shop array ID description",
						Type:        "string",
						ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
							"title": "Shop array ID title",
						}},
					}},
					"banana_ids": {Value: &openapi3.Schema{
						Description: "Banana array description",
						Type:        "array",
						ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
							"title": "Banana array title",
						}},
						Items: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Description: "Banana item description",
							Type:        "string",
							ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
								"title": "Banana item title",
							}},
						}},
					}},
				},
			}})

			schema := transformJSON(responseContent(root.Paths["/shop_arrays/{id}"].Get.Responses["200"]))
			Expect(schema).To(Equal(expectedSchema))
		})
	})

	Describe("Parameters", func() {
		It("GET multiple should contain common parameters", func() {
			expectedParameters := []openapi3.Parameter{{
				Description: "Property key name to sort results",
				In:          "query",
				Name:        "sort_key",
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Default: "id",
					Type:    "string",
				}},
			}, {
				Description: "Sort order",
				In:          "query",
				Name:        "sort_order",
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Default: "asc",
					Enum:    []interface{}{"asc", "desc"},
					Type:    "string",
				}},
			}, {
				Description: "Maximum number of results",
				In:          "query",
				Name:        "limit",
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Type: "integer",
				}},
			}, {
				Description: "Number of results to be skipped",
				In:          "query",
				Name:        "offset",
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Type: "integer",
				}},
			}}

			parameters := transformJSON(root.Paths["/shops"].Get.Parameters)
			for _, expectedParameter := range expectedParameters {
				expectedParameter := transformJSON(expectedParameter)
				Expect(parameters).To(ContainElement(expectedParameter))
			}
		})

		It("GET multiple should contain parameters based on properties", func() {
			expectedParameters := []openapi3.Parameter{{
				Description: "Filter results with id value",
				In:          "query",
				Name:        "id",
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Type: "string",
				}},
			}, {
				Description: "Filter results with banana_id value",
				In:          "query",
				Name:        "banana_id",
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Type: "string",
				}},
			}}

			parameters := transformJSON(root.Paths["/shops"].Get.Parameters)
			for _, expectedParameter := range expectedParameters {
				expectedParameter := transformJSON(expectedParameter)
				Expect(parameters).To(ContainElement(expectedParameter))
			}
		})
	})

	Describe("Responses", func() {
		withDummyContent := func(response *openapi3.ResponseRef) *openapi3.ResponseRef {
			response.Value.Content = openapi3.NewContentWithJSONSchema(&openapi3.Schema{})
			return response
		}

		It("GET multiple should contain response with custom header", func() {
			expectedResponse := transformJSON(withDummyContent(&openapi3.ResponseRef{Value: &openapi3.Response{
				Description: "Banana resource description",
				Headers: map[string]*openapi3.HeaderRef{
					"X-Total-Count": {Value: &openapi3.Header{
						Description: "The number of banana elements",
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: "integer",
						}},
					}},
				},
			}}))

			response := transformJSON(withDummyContent(root.Paths["/bananas"].Get.Responses["200"]))
			Expect(response).To(Equal(expectedResponse))
		})

		It("DELETE should not contain schema in response", func() {
			expectedResponse := transformJSON(openapi3.Response{
				Description: "banana get deleted",
			})

			response := transformJSON(root.Paths["/bananas/{id}"].Delete.Responses["204"])
			Expect(response).To(Equal(expectedResponse))
		})

		It("normal custom action", func() {
			expectedResponse := transformJSON(openapi3.Response{
				Description: "action shop response",
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: "object",
							Properties: map[string]*openapi3.SchemaRef{
								"foobar": {Value: &openapi3.Schema{
									Description: "Normal action foobar description",
									Type:        "string",
									ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
										"title": "Normal action foobar title",
									}},
								}},
							},
						}},
					},
				},
			})

			response := transformJSON(root.Paths["/shops/{id}/action"].Get.Responses["200"])
			Expect(response).To(Equal(expectedResponse))
		})

		It("WebSocket custom action", func() {
			expectedResponse := transformJSON(openapi3.Response{
				Description: "WebSocket protocol",
			})

			response := transformJSON(root.Paths["/shops/{id}/websocket"].Get.Responses["default"])
			Expect(response).To(Equal(expectedResponse))
		})
	})

	Describe("Request body", func() {
		It("POST multiple", func() {
			expectedRequest := transformJSON(&openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
				Required:    true,
				Description: "peach resource input",
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: "object",
							Properties: map[string]*openapi3.SchemaRef{
								"color": {Value: &openapi3.Schema{
									Description: "Peach color description",
									Type:        "string",
									ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
										"title": "Peach color title",
									}},
								}},
							},
							AdditionalPropertiesAllowed: boolean(false),
						}},
					},
				},
			}})

			request := transformJSON(root.Paths["/peaches"].Post.RequestBody)
			Expect(request).To(Equal(expectedRequest))
		})

		It("PUT single", func() {
			expectedRequest := transformJSON(&openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
				Required:    true,
				Description: "peach resource input",
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: "object",
							Properties: map[string]*openapi3.SchemaRef{
								"sweetness": {Value: &openapi3.Schema{
									Description: "Peach sweetness description",
									Type:        "string",
									ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
										"title": "Peach sweetness title",
									}},
								}},
							},
							AdditionalPropertiesAllowed: boolean(false),
						}},
					},
				},
			}})

			request := transformJSON(root.Paths["/peaches/{id}"].Put.RequestBody)
			Expect(request).To(Equal(expectedRequest))
		})
	})

	It("operations should contains tags based on resource_group", func() {
		multiple := root.Paths["/shops"]
		single := root.Paths["/shops/{id}"]
		operations := []*openapi3.Operation{
			multiple.Get,
			multiple.Post,
			single.Get,
			single.Put,
			single.Delete,
		}

		expectedTags := []string{"Resource Group"}
		for _, operation := range operations {
			Expect(operation.Tags).To(Equal(expectedTags))
		}
	})

	DescribeTable("openapi3 filter parameter parsing",
		func(param string, expectedIndentPrefix string, expectedArgs map[string]bool) {
			indentPrefix, args := toOpenAPIv3ParseParam(param)
			Expect(indentPrefix).To(Equal(expectedIndentPrefix))
			Expect(args).To(Equal(expectedArgs))
		},
		Entry("indent", "  ", "  ", map[string]bool{}),
		Entry("parameter", "foo", "", map[string]bool{"foo": true}),
		Entry("indent and parameters", " ,foo,bar", " ", map[string]bool{"foo": true, "bar": true}),
	)
})

func removeRef(i interface{}) {
	removeRefRecursive(reflect.ValueOf(i))
}

func removeRefRecursive(value reflect.Value) {
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Ptr:
		removeRefRecursive(value.Elem())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i += 1 {
			if value.Type().Field(i).Name == "Ref" {
				value.Field(i).SetString("")
			}
			removeRefRecursive(value.Field(i))
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i += 1 {
			removeRefRecursive(value.Index(i))
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			removeRefRecursive(value.MapIndex(key))
		}
	}
}

// converts object to Go equivalent of JSON representation, which allows comparing them
// e.g. whether two objects are equal, array contains the element, etc.
func transformJSON(value interface{}) interface{} {
	// wrapping in some other object, allows arbitrary object to be marshaled and unmarshaled
	buf, err := json.Marshal([]interface{}{value})
	Expect(err).NotTo(HaveOccurred())
	result := []interface{}{}
	err = json.Unmarshal(buf, &result)
	Expect(err).NotTo(HaveOccurred())
	return result[0]
}

func command(cmd cli.Command, args ...string) []byte {
	app := cli.NewApp()
	app.Commands = []cli.Command{cmd}
	output, err := captureStdout(func() {
		Expect(app.Run(append([]string{"gohan", cmd.Name}, args...))).To(Succeed())
	})
	Expect(err).NotTo(HaveOccurred())
	return output
}

func captureStdout(f func()) ([]byte, error) {
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	os.Stdout = w
	defer func() {
		os.Stdout = stdout
	}()
	go func() {
		defer GinkgoRecover()
		defer func() { _ = w.Close() }()
		f()
	}()
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func number(number float64) *float64 {
	return &number
}

func boolean(boolean bool) *bool {
	return &boolean
}
