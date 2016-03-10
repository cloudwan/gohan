package cli

import (
	"encoding/json"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"path"
	"regexp"

	"github.com/flosch/pongo2"
)

func deleteGohanExtendedProperties(property map[string]interface{}) {
	delete(property, "unique")
	delete(property, "permission")
	delete(property, "relation")
	delete(property, "relation_property")
}

func toSwagger(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.Interface()
	m := i.(map[string]interface{})
	switch properties := m["properties"].(type) {
	case map[string]interface{}:
		for _, value := range properties {
			deleteGohanExtendedProperties(value.(map[string]interface{}))
		}
		delete(properties, "propertiesOrder")
	case map[string]map[string]interface{}:
		for _, value := range properties {
			deleteGohanExtendedProperties(value)
		}
		delete(properties, "propertiesOrder")
	}
	if list, ok := m["required"].([]string); ok {
		if len(list) == 0 {
			delete(m, "required")
		}
	}
	delete(m, "propertiesOrder")
	data, _ := json.MarshalIndent(i, param.String(), "    ")
	return pongo2.AsValue(string(data)), nil
}

func toSwaggerPath(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.String()
	r := regexp.MustCompile(":([^/]+)")
	return pongo2.AsValue(r.ReplaceAllString(i, "{$1}")), nil
}

func init() {
	pongo2.RegisterFilter("swagger", toSwagger)
	pongo2.RegisterFilter("swagger_path", toSwaggerPath)
}

func getTemplateCommand() cli.Command {
	return cli.Command{
		Name:        "template",
		ShortName:   "template",
		Usage:       "Convert gohan schema using pongo2 template",
		Description: "Convert gohan schema using pongo2 template",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: "etc/gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: "template, t", Value: "embed://etc/templates/swagger.tmpl", Usage: "Template File"},
			cli.StringFlag{Name: "output, o", Value: "swagger.json", Usage: "Output file"},
		},
		Action: func(c *cli.Context) {
			manager := schema.GetManager()
			configFile := c.String("config-file")
			outputFile := c.String("output")
			config := util.GetConfig()
			err := config.ReadConfig(configFile)
			if err != nil {
				util.ExitFatal(err)
				return
			}
			template := c.String("template")
			templateCode, err := util.GetContent(template)
			if err != nil {
				util.ExitFatal(err)
				return
			}
			pwd, _ := os.Getwd()
			os.Chdir(path.Dir(configFile))
			schemaFiles := config.GetStringList("schemas", nil)
			if schemaFiles == nil {
				util.ExitFatal("No schema specified in configuraion")
			} else {
				err = manager.LoadSchemasFromFiles(schemaFiles...)
				if err != nil {
					util.ExitFatal(err)
					return
				}
			}
			schemas := manager.OrderedSchemas()

			if err != nil {
				util.ExitFatal(err)
				return
			}
			tpl, err := pongo2.FromString(string(templateCode))
			if err != nil {
				util.ExitFatal(err)
				return
			}
			output, err := tpl.Execute(pongo2.Context{"schemas": schemas})
			if err != nil {
				util.ExitFatal(err)
				return
			}
			os.Chdir(pwd)
			ioutil.WriteFile(outputFile, []byte(output), os.ModePerm)
		},
	}
}
