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
	"io/ioutil"
	"os"
	"time"

	"github.com/cloudwan/gohan/schema"
)

// Cache ...
type Cache struct {
	Expire  time.Time
	Schemas []*schema.Schema
}

func (gohanClientCLI *GohanClientCLI) getCachedSchemas() ([]*schema.Schema, error) {
	rawCache, err := ioutil.ReadFile(gohanClientCLI.opts.cachePath)
	if err != nil {
		return gohanClientCLI.getSchemas()
	}
	cache := Cache{}
	err = json.Unmarshal(rawCache, &cache)
	if err != nil {
		return gohanClientCLI.getSchemas()
	}
	if time.Now().After(cache.Expire) {
		return gohanClientCLI.getSchemas()
	}
	return cache.Schemas, nil
}

func (gohanClientCLI *GohanClientCLI) setCachedSchemas() error {
	cache := Cache{
		Expire:  time.Now().Add(gohanClientCLI.opts.cacheTimeout),
		Schemas: gohanClientCLI.schemas,
	}
	rawCache, _ := json.Marshal(cache)
	err := ioutil.WriteFile(gohanClientCLI.opts.cachePath, rawCache, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
