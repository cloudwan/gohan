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
