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
	"reflect"
	"strings"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
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
		return &Transaction{tx}, true
	default:
		panic(fmt.Sprintf("Unknown transaction type in context: %+v", ctxTx))
	}
}

// NewUUID create a new unique ID
func (util *Util) NewUUID() string {
	return uuid.NewV4().String()
}

func (util *Util) GetTransaction(context goext.Context) (goext.ITransaction, bool) {
	return contextGetTransaction(context)
}

func (util *Util) Clone() *Util {
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
	resource := reflect.New(reflect.TypeOf(rawResource))
	if err := resourceFromMap(context, resource); err != nil {
		return nil, err
	}
	return resource.Interface(), nil
}

// expects primitive or allocated pointer to struct
func resourceFromMap(context map[string]interface{}, resource reflect.Value) error {
	if isPrimitiveKind(resource.Kind()) {
		resource.Set(reflect.ValueOf(context))
		return nil
	}
	if resource.Kind() == reflect.Interface {
		resource = resource.Elem()
	}
	if resource.Kind() == reflect.Ptr {
		if context == nil && resource.IsNil() {
			return nil
		} else if context != nil && resource.IsNil() {
			resource.Set(reflect.New(resource.Type().Elem()))
			return resourceFromMap(context, resource.Elem())
		}
		return resourceFromMap(context, resource.Elem())

	}
	for i := 0; i < resource.NumField(); i++ {
		field := resource.Field(i)
		fieldType := resource.Type().Field(i)
		if _, isMaybeStateField := field.Interface().(goext.Maybe); isMaybeStateField {
			field.Set(reflect.ValueOf(goext.Maybe{MaybeState: goext.MaybeValue}))
			continue
		}
		propertyName := strings.Split(fieldType.Tag.Get("json"), ",")[0]
		if propertyName == "" {
			continue
		}
		mapValue, mapValueExists := context[propertyName]
		kind := fieldType.Type.Kind()

		if kind == reflect.Interface {
			if field.IsNil() && mapValue == nil {
				continue
			}
			field.Set(reflect.ValueOf(mapValue))
		} else if isStructureMaybeType(field) {
			if !mapValueExists {
				setMaybeState(field, goext.MaybeUndefined)
			} else if mapValue == nil {
				setMaybeState(field, goext.MaybeNull)
			} else {
				field.Set(reflect.ValueOf(reflect.New(field.Type()).Elem().Interface()))
				if err := resourceFromMap(mapValue.(map[string]interface{}), field); err != nil {
					return err
				}
			}
		} else if isPrimitiveMaybeType(field) {
			if kind == reflect.Ptr {
				if !mapValueExists {
					continue
				}
				field.Set(reflect.New(fieldType.Type.Elem()))
				field = field.Elem()
			}
			if err := primitiveMaybeFromMap(context, propertyName, field); err != nil {
				return err
			}
		} else if kind == reflect.Struct || kind == reflect.Ptr {
			if mapValue != nil {
				field.Set(reflect.ValueOf(reflect.New(field.Type()).Elem().Interface()))
				if err := resourceFromMap(mapValue.(map[string]interface{}), field); err != nil {
					return err
				}
			}
		} else if kind == reflect.Slice {
			if err := sliceToMap(context, propertyName, field); err != nil {
				return err
			}
		} else {
			if err := assignMapValueToField(mapValue, propertyName, field); err != nil {
				return err
			}
		}
	}

	return nil
}

func assignMapValueToField(mapValue interface{}, fieldName string, field reflect.Value) error {
	value := reflect.ValueOf(mapValue)
	if value.IsValid() {
		if value.Type() == field.Type() {
			field.Set(value)
		} else {
			if field.Kind() == reflect.Int && value.Kind() == reflect.Float64 { // reflect treats number(N, 0) as float
				field.SetInt(int64(value.Float()))
			} else {
				return errors.Errorf("invalid type of '%s' field (%s, expecting %s)", fieldName, value.Kind(), field.Kind())
			}
		}
	}
	return nil
}

func primitiveMaybeFromMap(context map[string]interface{}, fieldName string, field reflect.Value) error {
	if mapValue, ok := context[fieldName]; !ok {
		// do nothing, undefined is default value
	} else if mapValue == nil {
		field.FieldByName("MaybeState").SetInt(int64(goext.MaybeNull))
	} else {
		field.FieldByName("MaybeState").SetInt(int64(goext.MaybeValue))
		setPrimitiveMaybe(field, mapValue)
	}
	return nil
}

func setPrimitiveMaybe(field reflect.Value, mapValue interface{}) {
	switch field.Interface().(type) {
	case goext.MaybeString:
		field.Set(reflect.ValueOf(goext.MakeString(mapValue.(string))))
	case goext.MaybeInt:
		switch mapValue.(type) {
		case int:
			field.Set(reflect.ValueOf(goext.MakeInt(mapValue.(int))))
		case float64:
			field.Set(reflect.ValueOf(goext.MakeInt(int(mapValue.(float64)))))
		}
	case goext.MaybeBool:
		field.Set(reflect.ValueOf(goext.MakeBool(mapValue.(bool))))
	case goext.MaybeFloat:
		field.Set(reflect.ValueOf(goext.MakeFloat(mapValue.(float64))))
	}
}

func sliceToMap(context map[string]interface{}, fieldName string, field reflect.Value) error {
	v, ok := context[fieldName]
	if !ok {
		return nil
	}

	elemType := field.Type().Elem()
	sliceElems := 0
	interfaces := false
	structures := false
	switch v.(type) {
	case []map[string]interface{}:
		sliceElems = len(v.([]map[string]interface{}))
		if elemType.Kind() == reflect.Ptr || elemType.Kind() == reflect.Struct {
			structures = true
		}
	case []interface{}:
		sliceElems = len(v.([]interface{}))
		interfaces = true
	default:
		val := reflect.ValueOf(v)
		if !val.IsValid() {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}
		sliceElems = val.Len()
	}
	field.Set(reflect.MakeSlice(field.Type(), sliceElems, sliceElems))
	field.SetLen(sliceElems)
	for i := 0; i < sliceElems; i++ {
		elem := field.Index(i)
		nestedField := reflect.New(elemType).Elem()
		if structures {
			if err := resourceFromMap(v.([]map[string]interface{})[i], nestedField); err != nil {
				return err
			}
		} else if interfaces {
			nestedValue := v.([]interface{})[i]
			if nestedMap, ok := nestedValue.(map[string]interface{}); ok && elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
				if err := resourceFromMap(nestedMap, nestedField); err != nil {
					return err
				}
			} else {
				nestedField.Set(reflect.ValueOf(nestedValue))
			}
		} else {
			val := reflect.ValueOf(v)
			nestedField.Set(val.Index(i))
		}
		elem.Set(nestedField)
	}

	return nil
}

// ResourceToMap converts structure representation of the resource to mapped representation
func (util *Util) ResourceToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("json")
	structMap := mapper.TypeMap(reflect.TypeOf(resource))
	resourceValue := reflect.ValueOf(resource).Elem()

	for fieldName, fi := range structMap.Names {
		if len(fi.Index) != 1 {
			continue
		}

		v := resourceValue.FieldByIndex(fi.Index)
		val := v.Interface()
		if fieldName == "id" && v.String() == "" {
			id := uuid.NewV4().String()
			fieldsMap[fieldName] = id
			v.SetString(id)
		} else if isPrimitiveMaybeType(v) {
			primitiveMaybeToMap(fieldsMap, fieldName, v)
		} else if isStructureMaybeType(v) {
			switch getMaybeState(v) {
			case goext.MaybeUndefined:
				// nothing
			case goext.MaybeNull:
				fieldsMap[fieldName] = nil
			case goext.MaybeValue:
				fieldsMap[fieldName] = util.ResourceToMap(v.Addr().Interface())
			}
		} else if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				if v.Type().Elem().Kind() != reflect.Struct {
					fieldsMap[fieldName] = nil
				}
			} else {
				rv := util.ResourceToMap(val)
				fieldsMap[fieldName] = rv
			}
		} else if v.Kind() == reflect.Slice {
			if !v.IsNil() {
				fieldsMap[fieldName] = util.sliceToMap(v)
			}
		} else if v.Kind() == reflect.Struct {
			fieldsMap[fieldName] = util.ResourceToMap(v.Addr().Interface())
		} else if v.Kind() == reflect.String {
			if v.String() != "" {
				fieldsMap[fieldName] = val
			}
		} else {
			fieldsMap[fieldName] = val
		}
	}

	return fieldsMap
}

func (util *Util) allSliceElemsAreMappable(v reflect.Value) bool {
	for i := 0; i < v.Len(); i++ {
		if !isMappableType(v.Index(i)) {
			return false
		}
	}
	return true
}

func (util *Util) mappableValueToMap(v reflect.Value) map[string]interface{} {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() == reflect.Map {
		return v.Interface().(map[string]interface{})
	} else {
		return util.ResourceToMap(v.Interface())
	}
}

func (util *Util) sliceToMapWithPrimitiveElems(v reflect.Value) interface{} {
	slice := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		slice[i] = v.Index(i).Interface()
	}
	return slice
}

func (util *Util) sliceToMapWithAnyElems(v reflect.Value) interface{} {
	slice := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Elem()
		if isMappableType(elem) {
			slice[i] = util.mappableValueToMap(elem)
		} else if elem.Kind() == reflect.Slice && !elem.IsNil() {
			slice[i] = util.sliceToMap(elem)
		} else {
			slice[i] = elem.Interface()
		}
	}
	return slice
}

func (util *Util) sliceToMapWithMappableElems(v reflect.Value) interface{} {
	slice := make([]map[string]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if isNullableKind(elem.Kind()) {
			if !elem.IsNil() {
				slice[i] = util.mappableValueToMap(elem)
			} else {
				slice[i] = nil
			}
		} else if elem.Kind() == reflect.Struct {
			slice[i] = util.ResourceToMap(elem.Addr().Interface())
		}
	}
	return slice
}

func (util *Util) sliceToMap(v reflect.Value) interface{} {
	elemKind := v.Type().Elem().Kind()
	if isPrimitiveKind(elemKind) {
		return util.sliceToMapWithPrimitiveElems(v)
	}
	if elemKind == reflect.Interface && !util.allSliceElemsAreMappable(v) {
		return util.sliceToMapWithAnyElems(v)
	} else {
		return util.sliceToMapWithMappableElems(v)
	}
}

func isStructureMaybeType(v reflect.Value) bool {
	if v.Kind() != reflect.Struct {
		return false
	}
	_, hasMaybeStateField := v.Type().FieldByName("MaybeState")
	if hasMaybeStateField && !isPrimitiveMaybeType(v) {
		return true
	}
	return false
}

func isMappableType(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Map:
		return true
	case reflect.Struct:
		return true
	case reflect.Interface:
		return isMappableType(v.Elem())
	case reflect.Ptr:
		return isMappableType(v.Elem())
	default:
		return false
	}
}

func isPrimitiveMaybeType(v reflect.Value) bool {
	switch v.Interface().(type) {
	case goext.MaybeString, goext.MaybeInt, goext.MaybeFloat, goext.MaybeBool:
		return true
	default:
		return false
	}
}
func getMaybeState(value reflect.Value) goext.MaybeState {
	return goext.MaybeState(value.FieldByName("MaybeState").Int())
}

func setMaybeState(v reflect.Value, state goext.MaybeState) {
	v.FieldByName("MaybeState").SetInt(int64(state))
}

func primitiveMaybeToMap(fieldsMap map[string]interface{}, fieldName string, value reflect.Value) {
	if value.Kind() == reflect.Interface {
		value = reflect.ValueOf(value.Interface())
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			panic("Nil pointers to Maybe types are not supported")
		}
		value = value.Elem()
	}

	switch getMaybeState(value) {
	case goext.MaybeUndefined:
		// nothing
	case goext.MaybeNull:
		fieldsMap[fieldName] = nil
	case goext.MaybeValue:
		emptyArgumentList := make([]reflect.Value, 0)
		fieldsMap[fieldName] = value.MethodByName("Value").Call(emptyArgumentList)[0].Interface()
	}
}

func isNullableKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Map, reflect.Slice, reflect.Interface, reflect.Ptr:
		return true
	}
	return false
}

func isPrimitiveKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	}
	return false
}

func GetSchemaID(schemaID interface{}) goext.SchemaID {
	if id, ok := schemaID.(goext.SchemaID); ok {
		return id
	}
	return goext.SchemaID(schemaID.(string))
}
