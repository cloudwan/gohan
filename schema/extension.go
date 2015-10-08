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

package schema

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/cloudwan/gohan/util"
)

//Extension is a small plugin for gohan
type Extension struct {
	ID, CodeType, URL string
	Code              string
	Path              *regexp.Regexp
}

//NewExtension returns new extension from object
func NewExtension(raw interface{}) (*Extension, error) {
	typeData := raw.(map[string](interface{}))
	extension := &Extension{}
	extension.ID, _ = typeData["id"].(string)
	extension.CodeType, _ = typeData["code_type"].(string)
	if _, ok := typeData["url"].(string); ok {
		url, err := url.Parse(typeData["url"].(string))
		if err != nil {
			return nil, err
		}
		if isURLRelative := url.Scheme == "file" && url.Host == "."; isURLRelative {
			workingDirectory, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			url.Host = ""
			url.Path = filepath.Join(workingDirectory, url.Path)
		}
		extension.URL = url.String()
	}
	extension.Code, _ = typeData["code"].(string)

	path, _ := typeData["path"].(string)
	match, err := regexp.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse regexp: %s", err)
	}

	extension.Path = match
	if extension.URL != "" {
		remoteCode, err := util.GetContent(extension.URL)
		extension.Code += string(remoteCode)
		if err != nil {
			return nil, fmt.Errorf("failed to load remote code: %s", err)
		}
	}
	return extension, nil
}

//Match checks if this path matches for extension
func (e *Extension) Match(path string) bool {
	return e.Path.MatchString(path)
}
