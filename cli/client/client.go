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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/common/extensions"

	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
)

var (
	log                 = l.NewLoggerForModule("gohan.cli.client")
	logOutput io.Writer = os.Stderr
)

const schemaListTemplate = `gohan client {{.ID}} # {{.Title}}
`

const schemaTemplate = `
  {{.Title}}
  -----------------------------------
  Description: {{.Description}}

  Properties: {{$properties := .JSONSchema.properties}}
  {{range $key := .JSONSchema.propertiesOrder}} - {{$key}} {{with index $properties $key}}[ {{.type}} ]: {{.title}} {{.description}}{{end}}
  {{end}}

  Commands:

  - List all {{.Title}} resources

    gohan client {{.ID}} list

  - Show a {{.Title}} resources

    gohan client {{.ID}} show [ID]

  - Create a {{.Title}} resources

    gohan client {{.ID}} create \
{{$propertiesOnCreate := .JSONSchemaOnCreate.properties}}{{range $key := .JSONSchema.propertiesOrder}}{{with index $propertiesOnCreate $key}}      --{{$key}} [ {{.type}} ] \
{{end}}{{end}}

  - Update {{.Title}} resources

    gohan client {{.ID}} set \
{{$propertiesOnUpdate := .JSONSchemaOnUpdate.properties}}{{range $key := .JSONSchema.propertiesOrder}}{{with index $propertiesOnUpdate $key}}      --{{$key}} [{{.type}} ] \
{{end}}{{end}}       [ID]

  - Delete {{.Title}} resources

  gohan client {{.ID}} delete [ID]

`

// GohanClientCLI ...
type GohanClientCLI struct {
	provider *gophercloud.ProviderClient
	schemas  []*schema.Schema
	commands []gohanCommand
	opts     *GohanClientCLIOpts
}

func setUpLogging(level l.Level) {
	l.SetUpBasicLogging(logOutput, l.CliFormat, "gohan.cli.client", level)
}

// ExecuteCommand ...
func (gohanClientCLI *GohanClientCLI) ExecuteCommand(command string, arguments []string) (string, error) {
	for _, c := range gohanClientCLI.commands {
		if c.Name == command {
			return c.Action(arguments)
		}
	}
	return gohanClientCLI.outputSubCommands(command)
}

//Output sub command helps
func (gohanClientCLI *GohanClientCLI) outputSubCommands(command string) (string, error) {
	schemas, err := gohanClientCLI.getSchemas()
	command = strings.TrimSpace(command)
	buf := new(bytes.Buffer)
	if err != nil {
		return "", err
	}
	//Output schema specific help
	for _, schema := range schemas {
		if command == schema.ID {
			tmpl, _ := template.New("schema").Parse(schemaTemplate)
			tmpl.Execute(buf, schema)
			return buf.String(), nil
		}
	}

	tmpl, _ := template.New("schema").Parse(schemaListTemplate)
	if command != "" {
		buf.WriteString("Command not found")
		return buf.String(), nil
	}
	for _, schema := range schemas {
		tmpl.Execute(buf, schema)
	}
	return buf.String(), nil
}

// NewGohanClientCLI GohanClientCLI constructor
func NewGohanClientCLI(opts *GohanClientCLIOpts) (*GohanClientCLI, error) {
	gohanClientCLI := GohanClientCLI{
		opts: opts,
	}
	setUpLogging(gohanClientCLI.opts.logLevel)

	provider, err := getProviderClient()
	if err != nil {
		return nil, err
	}
	gohanClientCLI.provider = provider

	if opts.authTokenID != "" {
		gohanClientCLI.provider.TokenID = opts.authTokenID
	}

	if opts.gohanEndpointURL == "" {
		gohanEndpointURL, err := gohanClientCLI.getGohanEndpointURL(provider)
		if err != nil {
			return nil, err
		}
		gohanClientCLI.opts.gohanEndpointURL = gohanEndpointURL
	}

	var schemas []*schema.Schema
	if gohanClientCLI.opts.cacheSchemas {
		schemas, err = gohanClientCLI.getCachedSchemas()
	} else {
		schemas, err = gohanClientCLI.getSchemas()
	}
	if err != nil {
		return nil, err
	}
	gohanClientCLI.schemas = schemas
	if gohanClientCLI.opts.cacheSchemas {
		err := gohanClientCLI.setCachedSchemas()
		if err != nil {
			return nil, fmt.Errorf("Error caching schemas: %v", err)
		}
	}

	gohanClientCLI.commands = gohanClientCLI.getCommands()

	return &gohanClientCLI, nil
}

func getProviderClient() (*gophercloud.ProviderClient, error) {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return nil, err
	}
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		if strings.Contains(err.Error(), "provide exactly one of Domain") {
			return nil, fmt.Errorf(envVariablesNotSetError, keystoneDomainIDKey, keystoneDomainNameKey)
		}
	}
	return provider, err
}

func (gohanClientCLI *GohanClientCLI) getGohanEndpointURL(provider *gophercloud.ProviderClient) (string, error) {
	endpointOpts := gophercloud.EndpointOpts{
		Type:         gohanClientCLI.opts.gohanServiceName,
		Region:       gohanClientCLI.opts.gohanRegion,
		Availability: gophercloud.AvailabilityAdmin,
	}
	endpoint, err := provider.EndpointLocator(endpointOpts)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(endpoint, "/"), nil
}

func (gohanClientCLI *GohanClientCLI) getSchemas() ([]*schema.Schema, error) {
	response := extensions.GetResult{}
	url := fmt.Sprintf("%s%s", gohanClientCLI.opts.gohanEndpointURL, gohanClientCLI.opts.gohanSchemaURL)
	gohanClientCLI.logRequest("GET", url, gohanClientCLI.provider.TokenID, nil)
	_, err := gohanClientCLI.provider.Get(url, &response.Body, nil)
	if err != nil {
		return nil, err
	}

	bodyMap := response.Body.(map[string]interface{})
	if _, ok := bodyMap["schemas"]; !ok {
		return nil, fmt.Errorf("No 'schemas' key in response JSON")
	}

	result := []*schema.Schema{}
	for _, rawSchema := range bodyMap["schemas"].([]interface{}) {
		schema, err := schema.NewSchemaFromObj(rawSchema)
		if err != nil {
			return nil, fmt.Errorf("Could not parse schemas: %v", err)
		}
		result = append(result, schema)
	}
	return result, nil
}

func (gohanClientCLI *GohanClientCLI) getSchemaByID(id string) (*schema.Schema, error) {
	for _, s := range gohanClientCLI.schemas {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, fmt.Errorf("Schema with ID '%s' not found", id)
}

func (gohanClientCLI *GohanClientCLI) logRequest(method, url, tokenID string, args map[string]interface{}) {
	log.Notice("Sent request: %s %s", method, url)
	log.Debug("X-Auth-Token: %s", tokenID)
	jsonArgs, _ := json.MarshalIndent(args, "", "    ")
	log.Info("Request body:\n %s", jsonArgs)
}

func (gohanClientCLI *GohanClientCLI) logResponse(status string, body interface{}) {
	log.Notice("Received response: %s", status)
	jsonBody, _ := json.MarshalIndent(body, "", "    ")
	log.Info("Response body:\n %s", jsonBody)
}
