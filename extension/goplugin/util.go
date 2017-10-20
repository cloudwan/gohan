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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/twinj/uuid"
)

type Util struct {
}

func contextGetTransaction(ctx goext.Context) (goext.ITransaction, bool) {
	ctxTx := ctx["transaction"]
	if ctxTx == nil {
		return nil, false
	}

	switch tx := ctxTx.(type) {
	case goext.ITransaction:
		return tx, true
	case transaction.Transaction:
		return &Transaction{tx, true}, true
	default:
		panic(fmt.Sprintf("Unknown transaction type in context: %+v", ctxTx))
	}
}

// NewUUID create a new unique ID
func (util *Util) NewUUID() string {
	return uuid.NewV4().String()
}

func (u *Util) GetTransaction(context goext.Context) (goext.ITransaction, bool) {
	return contextGetTransaction(context)
}

func (u *Util) Clone() *Util {
	return &Util{}
}

var controllers map[gomock.TestReporter]*gomock.Controller = make(map[gomock.TestReporter]*gomock.Controller)

func NewController(testReporter gomock.TestReporter) *gomock.Controller {
	ctrl := gomock.NewController(testReporter)
	controllers[testReporter] = ctrl
	return ctrl
}

func Finish(testReporter gomock.TestReporter) {
	controllers[testReporter].Finish()
}

// ResourceFromMapForType converts mapped representation to structure representation of the resource for given type
func (util *Util) ResourceFromMapForType(context map[string]interface{}, rawResource interface{}) (goext.Resource, error) {
	return resourceFromMap(context, reflect.TypeOf(rawResource))
}

func resourceFromMap(context map[string]interface{}, rawType reflect.Type) (res goext.Resource, err error) {
	resource := reflect.New(rawType)

	for i := 0; i < rawType.NumField(); i++ {
		field := resource.Elem().Field(i)
		fieldType := rawType.Field(i)
		propertyName := strings.Split(fieldType.Tag.Get("json"), ",")[0]
		if propertyName == "" {
			return nil, fmt.Errorf("missing tag 'json' for resource %s field %s", rawType.Name(), fieldType.Name)
		}
		kind := fieldType.Type.Kind()
		if strings.Contains(fieldType.Type.String(), "goext.Null") {
			if context[propertyName] == nil {
				field.FieldByName("Valid").SetBool(false)
			} else {
				field.FieldByName("Valid").SetBool(true)
				value := reflect.ValueOf(context[propertyName])
				field.FieldByName("Value").Set(value)
			}
		} else if kind == reflect.Struct || kind == reflect.Ptr || kind == reflect.Slice {
			mapJSON, err := json.Marshal(context[propertyName])
			if err != nil {
				return nil, err
			}
			newField := reflect.New(field.Type())
			fieldJSON := string(mapJSON)
			fieldInterface := newField.Interface()
			err = json.Unmarshal([]byte(fieldJSON), &fieldInterface)
			if err != nil {
				return nil, err
			}
			field.Set(newField.Elem())
		} else {
			value := reflect.ValueOf(context[propertyName])
			if value.IsValid() {
				if value.Type() == field.Type() {
					field.Set(value)
				} else {
					if field.Kind() == reflect.Int && value.Kind() == reflect.Float64 { // reflect treats number(N, 0) as float
						field.SetInt(int64(value.Float()))
					} else {
						return nil, fmt.Errorf("invalid type of '%s' field (%s, expecting %s)", propertyName, value.Kind(), field.Kind())
					}
				}
			}
		}
	}

	return resource.Interface(), nil
}

// ResourceToMap converts structure representation of the resource to mapped representation
func (util *Util) ResourceToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("json")
	structMap := mapper.TypeMap(reflect.TypeOf(resource))
	resourceValue := reflect.ValueOf(resource).Elem()

	for field, fi := range structMap.Names {
		if len(fi.Index) != 1 {
			continue
		}

		v := resourceValue.FieldByIndex(fi.Index)
		val := v.Interface()
		if field == "id" && v.String() == "" {
			id := uuid.NewV4().String()
			fieldsMap[field] = id
			v.SetString(id)
		} else if strings.Contains(v.Type().String(), "goext.Null") {
			valid := v.FieldByName("Valid").Bool()
			if valid {
				fieldsMap[field] = v.FieldByName("Value").Interface()
			} else {
				fieldsMap[field] = nil
			}
		} else if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				fieldsMap[field] = nil
			} else {
				fieldsMap[field] = util.ResourceToMap(val)
			}
		} else {
			fieldsMap[field] = val
		}
	}

	return fieldsMap
}
