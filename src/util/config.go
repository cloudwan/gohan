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
	"os"
	"path/filepath"
	"strings"
)

//Config stores configuration paramters for api server
type Config struct {
	config map[string]interface{}
}

var gohanConfig *Config

//GetConfig returns configuraion data
func GetConfig() *Config {
	if gohanConfig == nil {
		config := map[string]interface{}{}
		gohanConfig = &Config{
			config: config,
		}
	}
	return gohanConfig
}

//GetEnvMap reads environemnt vars and return key value
func GetEnvMap() map[string]string {
	envStrings := os.Environ()
	envMap := map[string]string{}
	for _, envKeyValue := range envStrings {
		keyValue := strings.Split(envKeyValue, "=")
		if len(keyValue) == 2 {
			key := keyValue[0]
			value := keyValue[1]
			envMap[key] = value
		}
	}
	return envMap
}

//ReadConfig reads data from config file
//Config file can be yaml or json file
func (config *Config) ReadConfig(path string) error {
	envMap := GetEnvMap()
	data, err := LoadTemplate(path, envMap)
	//(TODO) nati: verification for config data using json schema
	if err != nil {
		return err
	}
	for key, value := range data {
		if key == "include" {
			includePath := filepath.Join(filepath.Dir(path), value.(string))
			err := filepath.Walk(includePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				return config.ReadConfig(path)
			})
			if err != nil {
				return err
			}
		} else {
			config.config[key] = value
		}
	}
	return nil
}

//GetString returns string parameter from config
func (config *Config) GetString(key, defaultValue string) string {
	data := config.GetParam(key, defaultValue)
	dataString, ok := data.(string)
	if ok == false {
		return defaultValue
	}
	return dataString
}

//GetBool returns string parameter from config
func (config *Config) GetBool(key string, defaultValue bool) bool {
	data := config.GetParam(key, defaultValue)
	dataString, ok := data.(bool)
	if ok == false {
		return defaultValue
	}
	return dataString
}

//GetStringList returns string list parameter from config
func (config *Config) GetStringList(key string, defaultValue []string) []string {
	data := config.GetParam(key, defaultValue)
	if data == nil {
		return defaultValue
	}
	dataString, ok := data.([]interface{})
	if ok == false {
		return defaultValue
	}
	result := []string{}
	for _, value := range dataString {
		stringValue, ok := value.(string)
		if ok == false {
			return defaultValue
		}
		result = append(result, stringValue)
	}
	return result
}

//GetList returns list parameter from config
func (config *Config) GetList(key string, defaultValue []interface{}) []interface{} {
	data := config.GetParam(key, defaultValue)
	if data == nil {
		return defaultValue
	}
	dataInterface, ok := data.([]interface{})
	if ok == false {
		return defaultValue
	}
	return dataInterface
}

//GetParam returns parameter from config
func (config *Config) GetParam(key string, defaultValue interface{}) interface{} {
	key = "/" + key
	data, err := GetByJSONPointer(config.config, key)
	if err != nil {
		return defaultValue
	}
	return data
}

//(TODO) nati: Add more utility functions
