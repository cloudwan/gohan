package client

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/op/go-logging"
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
	incorrectVerbosityLevel   = "Incorrect verbosity level. Available level range %d %d"
	incorrectValueForArgument = "Incorrect value for '%s' enviroment variable, should be %s"

	defaultCachedSchemasPath = "/tmp/.cached-gohan-schemas"

	logLevelKey = "verbosity"
	logLevels   = []logging.Level{
		logging.WARNING,
		logging.NOTICE,
		logging.INFO,
		logging.DEBUG,
	}
	defaultLogLevel = logging.WARNING
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
	logLevel     logging.Level
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

	return &opts, nil
}
