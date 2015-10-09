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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/xeipuuv/gojsonpointer"
	"gopkg.in/yaml.v2"
)

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
	} else if strings.HasSuffix(file, ".yaml") {
		bytes, err = yaml.Marshal(data)
	}
	if err != nil {
		return err
	}
	ioutil.WriteFile(file, bytes, os.ModePerm)
	return nil
}

//LoadFile loads object from file. suffix of filepath will be
// used as file type. currently, json and yaml is supported
func LoadFile(filePath string) (map[string]interface{}, error) {
	bodyBuff, err := GetContent(filePath)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(filePath, ".json") {
		var document map[string]interface{}
		err = json.Unmarshal(bodyBuff, &document)
		return document, err
	} else if strings.HasSuffix(filePath, ".yaml") {
		var documentYAML map[interface{}]interface{}
		err = yaml.Unmarshal(bodyBuff, &documentYAML)
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
