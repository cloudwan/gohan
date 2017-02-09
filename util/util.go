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

package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"golang.org/x/crypto/ssh"

	"gopkg.in/yaml.v2"

	"github.com/xeipuuv/gojsonpointer"
)

//Counter represents atomic counter
type Counter struct {
	value int64
}

//NewCounter makes atomic counter
func NewCounter(value int64) *Counter {
	return &Counter{value: value}
}

//Add add value to the counter
func (counter *Counter) Add(value int64) {
	atomic.AddInt64(&counter.value, int64(value))
}

//Value get current value
func (counter *Counter) Value() int64 {
	return atomic.LoadInt64(&counter.value)
}

//GetByJSONPointer returns subdata of json using json pointer
func GetByJSONPointer(inData interface{}, key string) (interface{}, error) {
	pointer, err := gojsonpointer.NewJsonPointer(key)
	if err != nil {
		return nil, err
	}
	data, _, err := pointer.Get(inData)
	if err != nil {
		return nil, err
	}
	return data, nil

}

//SaveFile saves object to file. suffix of filepath will be
// used as file type. currently, json and yaml is supported
func SaveFile(file string, data interface{}) error {
	var bytes []byte
	var err error
	if strings.HasSuffix(file, ".json") {
		bytes, err = json.MarshalIndent(data, "", "    ")
	} else if strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml") {
		bytes, err = yaml.Marshal(data)

	}
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, bytes, os.ModePerm)
}

//LoadMap loads map object from file. suffix of filepath will be
// used as file type. currently, json and yaml is supported
func LoadMap(filePath string) (map[string]interface{}, error) {
	data, err := LoadFile(filePath)
	if err != nil {
		return nil, err
	}
	d, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data isn't map")
	}
	return d, nil
}

//LoadFile loads object from file. suffix of filepath will be
// used as file type. currently, json and yaml is supported
func LoadFile(filePath string) (interface{}, error) {
	bodyBuff, err := GetContent(filePath)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(filePath, ".json") {
		var document interface{}
		err = json.Unmarshal(bodyBuff, &document)
		return document, err
	} else if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		var documentYAML interface{}
		err = yaml.Unmarshal(bodyBuff, &documentYAML)
		if err != nil {
			return map[string]interface{}{}, err
		}
		document := DecodeYAMLLibObject(documentYAML)
		return document, nil
	}
	return nil, err
}

//LoadTemplate loads object from file. suffix of filepath will be
// used as file type. currently, json and yaml is supported
func LoadTemplate(filePath string, params interface{}) (map[string]interface{}, error) {
	templateSource, err := GetContent(filePath)
	t := template.Must(template.New("tmpl").Parse(string(templateSource[:])))
	outputBuffer := bytes.NewBuffer(make([]byte, 0, 100))
	t.Execute(outputBuffer, params)
	data := outputBuffer.Bytes()
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(filePath, ".json") {
		var document map[string]interface{}
		err = json.Unmarshal(data, &document)
		return document, err
	} else if strings.HasSuffix(filePath, ".yaml") {
		var documentYAML map[interface{}]interface{}
		err = yaml.Unmarshal(data, &documentYAML)
		if err != nil {
			return map[string]interface{}{}, err
		}
		document := DecodeYAMLLibObject(documentYAML)
		return document.(map[string]interface{}), nil
	}
	return nil, err
}

//DecodeYAMLLibObject decodes interface format
//yaml.v2 lib returns map as map[interface{}]interface{} while gohan lib uses map[string]interface{}
//so we need to convert it here
func DecodeYAMLLibObject(yamlData interface{}) interface{} {
	yamlMap, ok := yamlData.(map[interface{}]interface{})
	if ok {
		mapData := map[string]interface{}{}
		for key, value := range yamlMap {
			mapData[key.(string)] = DecodeYAMLLibObject(value)
		}
		return mapData
	}
	yamlList, ok := yamlData.([]interface{})
	if ok {
		mapList := []interface{}{}
		for _, value := range yamlList {
			mapList = append(mapList, DecodeYAMLLibObject(value))
		}
		return mapList
	}
	return yamlData
}

//GetContent loads file from remote or local
func GetContent(url string) ([]byte, error) {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		resp, _ := http.Get(url)
		defer resp.Body.Close()
		content, err := ioutil.ReadAll(resp.Body)
		return content, err
	}
	if strings.HasPrefix(url, "embed://") {
		url = strings.TrimPrefix(url, "embed://")
		return Asset(url)
	}
	if strings.HasPrefix(url, "file://") {
		url = strings.TrimPrefix(url, "file://")
	}
	content, err := ioutil.ReadFile(url)
	return content, err
}

// TempFile creates a temporary file with the specified prefix and suffix
func TempFile(dir string, prefix string, suffix string) (*os.File, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	name := filepath.Join(dir, fmt.Sprint(prefix, time.Now().UnixNano(), suffix))
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
}

// ExitFatal ...
func ExitFatal(a ...interface{}) {
	fmt.Println(a...)
	os.Exit(1)
}

// ExitFatalf ...
func ExitFatalf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
	os.Exit(1)
}

// GetSortedKeys returns sorted keys of map[string]interface{}
func GetSortedKeys(object map[string]interface{}) []string {
	keys := []string{}
	for key := range object {
		keys = append(keys, key)
	}
	sort.Sort(stringSlice(keys))
	return keys
}

type stringSlice []string

func (p stringSlice) Len() int           { return len(p) }
func (p stringSlice) Less(i, j int) bool { return strings.ToLower(p[i]) < strings.ToLower(p[j]) }
func (p stringSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

//PublicKeyFile read file and return ssh public keys
func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

//Match if a contrains b or a == b
func Match(a interface{}, b interface{}) bool {
	switch a.(type) {
	case string:
		return a == b
	case []interface{}:
		for _, item := range a.([]interface{}) {
			if item == b {
				return true
			}
		}
	}
	return false
}

//ContainsString checks if we have a string in a string list
func ContainsString(list []string, value string) bool {
	if list == nil {
		return false
	}
	for _, item := range list {
		if value == item {
			return true
		}
	}
	return false
}

//ExtendStringList appends a value if it isn't in base list
func ExtendStringList(original []string, extension []string) []string {
	extended := make([]string, len(original))
	copy(extended, original)
	for _, item := range extension {
		if !ContainsString(extended, item) {
			extended = append(extended, item)
		}
	}
	return extended
}

//ExtendMap extends a map from existing one
func ExtendMap(original map[string]interface{}, extension map[string]interface{}) map[string]interface{} {
	extended := map[string]interface{}{}
	for key, value := range original {
		extended[key] = value
	}
	for key, value := range extension {
		if _, ok := extended[key]; !ok {
			extended[key] = value
		}
	}
	return extended
}

//MaybeString returns "" if data is nil
func MaybeString(data interface{}) string {
	stringData, _ := data.(string)
	return stringData
}

//MaybeStringList returns empty list if data is nil
func MaybeStringList(data interface{}) []string {
	stringList, ok := data.([]string)
	if !ok {
		stringList = []string{}
		if interfaceList, ok := data.([]interface{}); ok {
			for _, value := range interfaceList {
				if stringValue, ok := value.(string); ok {
					stringList = append(stringList, stringValue)
				}
			}
		}
	}
	return stringList
}

//MaybeMap returns empty map if data is nil
func MaybeMap(data interface{}) map[string]interface{} {
	mapValue, ok := data.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return mapValue
}

//MaybeList tries cast to list otherwise returns empty object
func MaybeList(value interface{}) []interface{} {
	res, ok := value.([]interface{})
	if !ok {
		return []interface{}{}
	}
	return res
}

//MaybeInt tries cast to int otherwise returns 0
func MaybeInt(value interface{}) int {
	res, _ := value.(int)
	return res
}
