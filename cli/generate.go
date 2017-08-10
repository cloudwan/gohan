package cli

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/codegangsta/cli"
)

//TemplateDef template def defines file
type TemplateDef struct {
	Type         string `yaml:"type"`          // All, ResourceGroup, Resource
	OutputPath   string `yaml:"output_path"`   // This could be template string. "dir/{{Resource.ID}}".go
	TemplatePath string `yaml:"template_path"` // Path to template
}

func getGenerateCommand() cli.Command {
	return cli.Command{
		Name:      "generate",
		ShortName: "gen",
		Usage:     "Generate ServerSide Code",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "template, t", Value: "", Usage: "Application template path"},
			cli.StringFlag{Name: "templates", Value: "", Usage: "Template Configuraion"},
			cli.StringFlag{Name: "config-file, c", Value: "./gohan.yaml", Usage: "Gohan config file"},
			cli.StringFlag{Name: "output, o", Value: ".", Usage: "Dir of output"},
			cli.StringFlag{Name: "package, p", Value: "gen", Usage: "Package Name"},
			cli.StringFlag{Name: "dbname, d", Value: "gohan", Usage: "DB Name"},
			cli.BoolFlag{Name: "resetdb", Usage: "Reset Database on create"},
		},
		Action: gohanGenerate,
	}
}

func ensureDatabase(dbName string) error {
	config := util.GetConfig()
	databaseType := config.GetString("database/type", "mysql")
	databaseConnection := os.Getenv("DATABASE_CONNECTION")
	if databaseConnection == "" {
		databaseConnection = config.GetString("database/connection", "")
	}
	db, err := sql.Open(databaseType, databaseConnection)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("drop database if exists " + dbName)
	if err != nil {
		return err
	}
	_, err = db.Exec("create database " + dbName)
	if err != nil {
		return err
	}
	return nil
}

func dropDatabase(dbName string) error {
	config := util.GetConfig()
	databaseType := config.GetString("database/type", "mysql")
	databaseConnection := os.Getenv("DATABASE_CONNECTION")
	if databaseConnection == "" {
		databaseConnection = config.GetString("database/connection", "")
	}
	db, err := sql.Open(databaseType, databaseConnection)
	if err != nil {
		return nil
	}
	defer db.Close()
	_, err = db.Exec("drop database " + dbName)
	if err != nil {
		return nil
	}
	return nil
}

func gohanGenerate(c *cli.Context) {
	path := c.String("output")
	template := c.String("template")
	templates := c.String("templates")
	configFile := c.String("config-file")
	packageName := c.String("package")
	dbName := c.String("dbname")
	codeDir := filepath.Join(path, packageName)
	etcDir := filepath.Join(path, "etc")
	dbDir := filepath.Join(etcDir, "db")
	resetDB := c.Bool("resetdb")
	migrationDir := filepath.Join(dbDir, "migrations")
	schemaPath := filepath.Join(etcDir, "schema.json")
	manager := schema.GetManager()
	config := util.GetConfig()
	config.ReadConfig(configFile)
	schemaFiles := config.GetStringList("schemas", nil)
	if schemaFiles == nil {
		log.Fatal("No schema specified in configuraion")
		return
	}
	if err := manager.LoadSchemasFromFiles(schemaFiles...); err != nil {
		log.Fatal(err)
		return
	}
	// Genrating schema json
	log.Info("Genrating: schema json")

	list := []interface{}{}

	for _, schema := range manager.OrderedSchemas() {
		if schema.IsAbstract() {
			continue
		}
		if schema.Metadata["type"] == "metaschema" {
			continue
		}
		s := schema.JSON()
		s["url"] = schema.URL
		list = append(list, s)
	}
	os.Mkdir(etcDir, 0777)
	os.Mkdir(dbDir, 0777)
	os.Mkdir(migrationDir, 0777)
	execCommand(fmt.Sprintf("rm %s/*_init_schema.sql", migrationDir))
	if resetDB {
		err := ensureDatabase(dbName)
		if err != nil {
			log.Error("Failed to reset database", err)
		}
	}
	execCommand(
		fmt.Sprintf(
			"gohan migrate init --config-file %s", configFile))
	execCommand(
		fmt.Sprintf(
			"gohan migrate up --config-file %s", configFile))
	// Running sqlboiler
	execCommand("sqlboiler mysql")

	util.SaveFile(schemaPath, map[string]interface{}{
		"schemas": list,
	})
	execCommand(
		fmt.Sprintf(
			"go-bindata -pkg %s -o %s/go-bindata.go %s", packageName, codeDir, schemaPath))

	//Generating application code
	log.Info("Generating: application code")
	if templates != "" {
		templateConfig := []*TemplateDef{}
		templateDir := filepath.Dir(templates)
		data, err := ioutil.ReadFile(templates)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		err = yaml.Unmarshal([]byte(data), &templateConfig)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		for _, templateDef := range templateConfig {
			flag := ""
			templatePath := filepath.Join(templateDir, templateDef.TemplatePath)
			switch templateDef.Type {
			case "resource":
				flag = "--split-by-resource true"
			case "group":
				flag = "--split-by-resource-group true"
			}
			execCommand(
				fmt.Sprintf(
					"gohan template --config-file %s --template %s %s --output-path %s",
					configFile, templatePath,
					flag, filepath.Join(codeDir, templateDef.OutputPath),
				))
		}
		execCommand(
			fmt.Sprintf(
				"goimports -w %s/*.go", codeDir))
	}
	if template != "" {
		execCommand(
			fmt.Sprintf(
				"gohan template --config-file %s --template %s | grep -v '^\\s*$' > %s/base_controller.go", configFile, template, packageName))
		execCommand(
			fmt.Sprintf(
				"goimports -w %s/base_controller.go", codeDir))
	}

}

func execCommand(command string) {
	output, err := exec.Command("sh", "-c", command).Output()
	log.Info("Running: %s", command)
	outputStr := string(output[:])
	if outputStr != "" {
		log.Info("Output: %s", outputStr)
	}
	if err != nil {
		log.Error("Error: %s", err)
	}
}
