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
	"fmt"
	"os"
	"strconv"
	"time"

	l "github.com/cloudwan/gohan/log"
)

var (
	gohanEndpointURLKey       = "GOHAN_ENDPOINT_URL"
	gohanServiceNameKey       = "GOHAN_SERVICE_NAME"
	gohanRegionKey            = "GOHAN_REGION"
	gohanSchemaURLKey         = "GOHAN_SCHEMA_URL"
	keystoneDomainNameKey     = "OS_DOMAIN_NAME"
	keystoneDomainIDKey       = "OS_DOMAIN_ID"
	keystoneTokenIDKey        = "OS_TOKEN_ID"
	cacheSchemasKey           = "GOHAN_CACHE_SCHEMAS"
	cacheTimeoutKey           = "GOHAN_CACHE_TIMEOUT"
	cachePathKey              = "GOHAN_CACHE_PATH"
	envVariableNotSetError    = "Environment variable %v needs to be set"
	envVariablesNotSetError   = "Environment variable %v or %v needs to be set"
	incorrectOutputFormat     = "Incorrect output format. Available formats: %v"
	incorrectVerbosityLevel   = "Incorrect verbosity level. Available level range %d %d"
	incorrectValueForArgument = "Incorrect value for '%s' environment variable, should be %s"

	defaultCachedSchemasPath = "/tmp/.cached-gohan-schemas"

	outputFormatKey    = "output-format"
	outputFormatEnvKey = "GOHAN_OUTPUT_FORMAT"
	outputFormatTable  = "table"
	outputFormatJSON   = "json"
	outputFormats      = []string{outputFormatTable, outputFormatJSON}

	logLevelKey    = "verbosity"
	logLevelEnvKey = "GOHAN_VERBOSITY"
	logLevels      = []l.Level{
		l.WARNING,
		l.NOTICE,
		l.INFO,
		l.DEBUG,
	}
	defaultLogLevel = l.WARNING

	commonParams = map[string]struct{}{
		outputFormatKey: struct{}{},
		logLevelKey:     struct{}{},
	}
)

// GohanClientCLIOpts options for GohanClientCLI
type GohanClientCLIOpts struct {
	authTokenID string

	cacheSchemas bool
	cacheTimeout time.Duration
	cachePath    string

	gohanEndpointURL string
	gohanServiceName string
	gohanRegion      string
	gohanSchemaURL   string

	outputFormat string
	logLevel     l.Level
}

// NewOptsFromEnv creates new Opts for GohanClientCLI using env variables
func NewOptsFromEnv() (*GohanClientCLIOpts, error) {
	opts := GohanClientCLIOpts{
		outputFormat: outputFormatTable,
		cacheSchemas: true,
		cacheTimeout: 5 * time.Minute,
		cachePath:    defaultCachedSchemasPath,
		logLevel:     defaultLogLevel,
	}

	opts.gohanEndpointURL = os.Getenv(gohanEndpointURLKey)

	if opts.gohanEndpointURL == "" {
		gohanServiceName := os.Getenv(gohanServiceNameKey)
		if gohanServiceName == "" {
			return nil, fmt.Errorf(envVariableNotSetError, gohanServiceNameKey)
		}
		opts.gohanServiceName = gohanServiceName

		gohanRegion := os.Getenv(gohanRegionKey)
		if gohanRegion == "" {
			return nil, fmt.Errorf(envVariableNotSetError, gohanRegionKey)
		}
		opts.gohanRegion = gohanRegion
	}

	rawCacheSchemas := os.Getenv(cacheSchemasKey)
	if rawCacheSchemas != "" {
		cacheSchemas, err := strconv.ParseBool(rawCacheSchemas)
		if err != nil {
			return nil, fmt.Errorf(incorrectValueForArgument, cacheSchemasKey, "bool")
		}
		opts.cacheSchemas = cacheSchemas
	}

	rawCacheTimeout := os.Getenv(cacheTimeoutKey)
	if rawCacheTimeout != "" {
		cacheTimeout, err := time.ParseDuration(rawCacheTimeout)
		if err != nil {
			return nil, fmt.Errorf(incorrectValueForArgument, cacheTimeoutKey, "e.g. 1h20m5s")
		}
		opts.cacheTimeout = cacheTimeout
	}

	cachePath := os.Getenv(cachePathKey)
	if cachePath != "" {
		opts.cachePath = cachePath
	}

	authTokenID := os.Getenv(keystoneTokenIDKey)
	if authTokenID != "" {
		opts.authTokenID = authTokenID
	}

	gohanSchemaURL := os.Getenv(gohanSchemaURLKey)
	if gohanSchemaURL == "" {
		return nil, fmt.Errorf(envVariableNotSetError, gohanSchemaURLKey)
	}
	opts.gohanSchemaURL = gohanSchemaURL

	outputFormatOpt := os.Getenv(outputFormatEnvKey)
	if outputFormatOpt != "" {
		outputFormat, err := findOutputFormat(outputFormatOpt)
		if err != nil {
			return nil, err
		}
		opts.outputFormat = outputFormat
	}

	verbosity := os.Getenv(logLevelEnvKey)
	if verbosity != "" {
		logLevel, err := parseLogLevel(verbosity)
		if err != nil {
			return nil, err
		}
		opts.logLevel = logLevel
	}

	return &opts, nil
}

func findOutputFormat(formatOpt interface{}) (string, error) {
	for _, format := range outputFormats {
		if format == formatOpt {
			return format, nil
		}
	}
	return "", fmt.Errorf(incorrectOutputFormat, outputFormats)
}

func parseLogLevel(verbosityOpt interface{}) (l.Level, error) {
	for i, logLevel := range logLevels {
		if fmt.Sprint(i) == verbosityOpt {
			return logLevel, nil
		}
	}
	return l.WARNING, fmt.Errorf(incorrectVerbosityLevel, 0, len(logLevels)-1)
}
