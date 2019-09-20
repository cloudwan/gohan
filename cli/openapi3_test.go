package cli

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/cloudwan/gohan/schema"
	"github.com/getkin/kin-openapi/openapi3"
	. "github.com/onsi/ginkgo"
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

		expectedBananaSchema = &openapi3.SchemaRef{Value: &openapi3.Schema{
			Type: "object",
			Properties: map[string]*openapi3.SchemaRef{
				"id": {Value: &openapi3.Schema{
					Description: "Banana ID description",
					Type:        "string",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{
						"title": "Banana ID title",
					}},
				}},
			},
		}}
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
		Expect(root.Info, WithTransform(transformJSON, Equal(transformJSON(openapi3.Info{
			Title:       apiTitle,
			Version:     apiVersion,
			Description: apiDescription,
		}))))
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
	})

	Describe("Schemas", func() {
		It("should embed schemas from properties with relation", func() {
			expectedSchema := transformJSON(&openapi3.SchemaRef{Value: &openapi3.Schema{
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
					"banana_object": expectedBananaSchema,
				},
			}})

			schema := transformJSON(root.Components.Schemas["shop"])
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

			schema := transformJSON(root.Components.Schemas["shop_array"])
			Expect(schema).To(Equal(expectedSchema))
		})

		It("should include additionalProperties", func() {
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
		It("GET multiple should contain response with list of objects", func() {
			expectedResponse := transformJSON(openapi3.Response{
				Description: "Banana resource description",
				Headers: map[string]*openapi3.HeaderRef{
					"X-Total-Count": {Value: &openapi3.Header{
						Description: "The number of banana elements",
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: "integer",
						}},
					}},
				},
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type:  "array",
							Items: expectedBananaSchema,
						}},
					},
				},
			})

			response := transformJSON(root.Paths["/bananas"].Get.Responses["200"])
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
	Expect(err).NotTo(HaveOccurred())
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
