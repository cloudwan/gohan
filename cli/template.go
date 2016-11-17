package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/codegangsta/cli"

	"github.com/flosch/pongo2"
	"strings"
)

func deleteGohanExtendedProperties(node map[string]interface{}) {
	extendedProperties := [...]string{"unique", "permission", "relation",
		"relation_property", "view", "detail_view", "propertiesOrder",
		"on_delete_cascade", "indexed", "relationColumn" }

	for _, extendedProperty := range extendedProperties {
		delete(node, extendedProperty)
	}
}

func fixEnumDefaultValue(node map[string]interface{}) {
	if defaultValue, ok := node["default"]; ok {
		if enums, ok := node["enum"]; ok {
			if defaultValueStr, ok := defaultValue.(string); ok {
				enumsArr := util.MaybeStringList(enums)
				if !util.ContainsString(enumsArr, defaultValueStr) {
					delete(node, "default")
				}
			}
		}
	}
}

func removeEmptyRequiredList(node map[string]interface{}) {
	const requiredProperty = "required"

	if required, ok := node[requiredProperty]; ok {
		switch list := required.(type) {
		case []string:
			if len(list) == 0 {
				delete(node, requiredProperty)
			}
		case []interface{}:
			if len(list) == 0 {
				delete(node, requiredProperty)
			}
		}
	}
}

func removeNotSupportedFormat(node map[string]interface{}) {
	const formatProperty string = "format"
	var allowedFormats = []string { "uri", "uuid", "email", "int32", "int64", "float", "double",
				        "byte", "binary", "date", "date-time", "password" }

	if format, ok := node[formatProperty]; ok {
		if format, ok := format.(string); ok {
			if !util.ContainsString(allowedFormats, format) {
				delete(node, formatProperty)
			}
		}
	}
}

func fixPropertyTree(node map[string]interface{}) {

	deleteGohanExtendedProperties(node)
	fixEnumDefaultValue(node)
	removeEmptyRequiredList(node)
	removeNotSupportedFormat(node)

	for _, value := range node {
		switch childs := value.(type) {
		case map[string]interface{}:
			fixPropertyTree(childs)
		case map[string]map[string]interface{}:
			for _, value := range childs {
				fixPropertyTree(value)
			}
		}
	}

}

func toSwagger(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.Interface()
	m := i.(map[string]interface{})

	fixPropertyTree(m)

	data, _ := json.MarshalIndent(i, param.String(), "    ")
	return pongo2.AsValue(string(data)), nil
}

func toSwaggerPath(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.String()
	r := regexp.MustCompile(":([^/]+)")
	return pongo2.AsValue(r.ReplaceAllString(i, "{$1}")), nil
}

func hasIdParam(in *pongo2.Value, param *pongo2.Value) (*pongo2. Value, *pongo2.Error) {
	i := in.String()
	return pongo2.AsValue(strings.Contains(i, ":id")), nil
}

func init() {
	pongo2.RegisterFilter("swagger", toSwagger)
	pongo2.RegisterFilter("swagger_path", toSwaggerPath)
	pongo2.RegisterFilter("swagger_has_id_param", hasIdParam)
}

func doTemplate(c *cli.Context) {
	template := c.String("template")
	manager := schema.GetManager()
	configFile := c.String("config-file")
	config := util.GetConfig()
	err := config.ReadConfig(configFile)
	if err != nil {
		util.ExitFatal(err)
		return
	}
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
	fmt.Println(output)
}

func getTemplateCommand() cli.Command {
	return cli.Command{
		Name:        "template",
		ShortName:   "template",
		Usage:       "Convert gohan schema using pongo2 template",
		Description: "Convert gohan schema using pongo2 template",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: "template, t", Value: "", Usage: "Template File"},
		},
		Action: doTemplate,
	}
}

func getOpenAPICommand() cli.Command {
	return cli.Command{
		Name:        "openapi",
		ShortName:   "openapi",
		Usage:       "Convert gohan schema to OpenAPI",
		Description: "Convert gohan schema to OpenAPI",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: "template, t", Value: "embed://etc/templates/openapi.tmpl", Usage: "Template File"},
		},
		Action: doTemplate,
	}
}

func getMarkdownCommand() cli.Command {
	return cli.Command{
		Name:        "markdown",
		ShortName:   "markdown",
		Usage:       "Convert gohan schema to markdown doc",
		Description: "Convert gohan schema to markdown doc",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: "template, t", Value: "embed://etc/templates/markdown.tmpl", Usage: "Template File"},
		},
		Action: doTemplate,
	}
}

func getDotCommand() cli.Command {
	return cli.Command{
		Name:        "dot",
		ShortName:   "dot",
		Usage:       "Convert gohan schema to dot file for graphviz",
		Description: "Convert gohan schema to dot file for graphviz",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: "template, t", Value: "embed://etc/templates/dot.tmpl", Usage: "Template File"},
		},
		Action: doTemplate,
	}
}
