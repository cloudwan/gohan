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

package server

import (
	"fmt"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

func setupEditor(server *Server) {
	manager := schema.GetManager()
	config := util.GetConfig()
	editableSchemaFile := config.GetString("editable_schema", "")
	extension.RegisterGoCallback("handle_schema",
		func(event string, context map[string]interface{}) error {
			auth := context["auth"].(schema.Authorization)
			if event == "pre_list" {
				list := []interface{}{}
				total := 0
				for _, currentSchema := range manager.OrderedSchemas() {
					trimmedSchema, err := schema.GetSchema(currentSchema, auth)
					if err != nil {
						return err
					}
					s := trimmedSchema.Data()
					s["url"] = currentSchema.URL
					if trimmedSchema != nil {
						list = append(list, s)
						total = total + 1
					}
				}
				context["total"] = total
				context["response"] = map[string]interface{}{
					"schemas": list,
				}
				return nil
			} else if event == "pre_show" {
				ID := context["id"].(string)
				currentSchema, _ := manager.Schema(ID)
				object, _ := schema.GetSchema(currentSchema, auth)
				s := object.Data()
				s["url"] = currentSchema.URL
				context["response"] = map[string]interface{}{
					"schema": s,
				}
				return nil
			}
			if event != "pre_create" && event != "pre_update" && event != "pre_delete" {
				return nil
			}

			if editableSchemaFile == "" {
				return nil
			}

			ID := context["id"].(string)

			schemasInFile, err := util.LoadFile(editableSchemaFile)
			if err != nil {
				return nil
			}
			schemas := schemasInFile["schemas"].([]interface{})
			updatedSchemas := []interface{}{}
			var existingSchema map[string]interface{}
			for _, rawSchema := range schemas {
				s := rawSchema.(map[string]interface{})
				if s["id"] == ID {
					existingSchema = s
				} else {
					updatedSchemas = append(updatedSchemas, s)
				}
			}

			if event == "pre_create" {
				if existingSchema != nil {
					return fmt.Errorf("ID has been taken")
				}
				schemasInFile["schemas"] = append(updatedSchemas, context["resource"].(map[string]interface{}))
				context["response"] = map[string]interface{}{
					"schema": context["resource"],
				}
				context["exception"] = map[string]interface{}{
					"name":    "CustomException",
					"message": context["response"],
					"code":    201,
				}
			} else if event == "pre_update" {
				if existingSchema == nil {
					return fmt.Errorf("Not found or Update isn't supported for this schema")
				}
				for key, value := range context["resource"].(map[string]interface{}) {
					existingSchema[key] = value
				}
				schemasInFile["schemas"] = append(updatedSchemas, existingSchema)
				context["response"] = map[string]interface{}{
					"schema": context["resource"],
				}
				context["exception"] = map[string]interface{}{
					"name":    "CustomException",
					"message": context["response"],
					"code":    200,
				}

			} else if event == "pre_delete" {
				if existingSchema == nil {
					return fmt.Errorf("Not found or Delete isn't supported for this schema")
				}
				schemasInFile["schemas"] = updatedSchemas
				deletedSchema, ok := manager.Schema(ID)
				if ok {
					manager.UnRegisterSchema(deletedSchema)
				}
				context["exception"] = map[string]interface{}{
					"name":    "CustomException",
					"message": map[string]interface{}{"result": "deleted"},
					"code":    204,
				}
			}
			util.SaveFile(editableSchemaFile, schemasInFile)
			err = manager.LoadSchemaFromFile(editableSchemaFile)
			if err != nil {
				return err
			}
			server.initDB()
			server.resetRouter()
			server.mapRoutes()
			return nil
		})
}
