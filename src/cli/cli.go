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

package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"

	"github.com/cloudwan/gohan/cli/client"
	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/extension/framework"
	"github.com/cloudwan/gohan/server"
	"github.com/codegangsta/cli"
)

//Run execute main command
func Run(name, usage, version string) {
	app := cli.NewApp()
	app.Name = "gohan"
	app.Usage = "Gohan"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug, d", Usage: "Show debug messages"},
	}
	app.Before = parseGlobalOptions
	app.Commands = []cli.Command{
		getGohanClientCommand(),
		getValidateCommand(),
		getInitDbCommand(),
		getConvertCommand(),
		getServerCommand(),
		getTestExtesionsCommand(),
		getMigrateCommand(),
	}
	app.Run(os.Args)
}

func parseGlobalOptions(c *cli.Context) (err error) {
	//TODO(marcin) do it
	return nil
}

func getGohanClientCommand() cli.Command {
	return cli.Command{
		Name:            "client",
		Usage:           "Manage Gohan resources",
		SkipFlagParsing: true,
		HideHelp:        true,
		Description: `gohan client schema_id command [arguments...]

COMMANDS:
    list                List all resources
    show                Show resource details
    create              Create resource
    set                 Update resource
    delete              Delete resource

ARGUMENTS:
    There are two types of arguments:
        - named:
            they are in '--name value' format and should be specified directly after
            command name.
            If you want to pass JSON null value, it should be written as: '--name "<null>"'.

            Special named arguments:
                --output-format [json/table] - specifies in which format results should be shown
                --verbosity [0-3] - specifies how much debug info Gohan Client should show (default 0)
        - unnamed:
            they are in 'value' format and should be specified at the end of the line,
            after all named arguments. At the moment only 'id' argument in 'show',
            'set' and 'delete' commands are available in this format.

    Identifying resources:
        It is possible to identify resources with its ID and Name.

        Examples:
            gohan client network show network-id
            gohan client network show "Network Name"
            gohan client subnet create --network "Network Name"
            gohan client subnet create --network network-id
            gohan client subnet create --network_id network-id

CONFIGURATION:
    Configuration is available by environment variables:
        * Keystone username - OS_USERNAME
        * Keystone password - OS_PASSWORD
        * Keystone tenant name or tenant id - OS_TENANT_NAME or OS_TENANT_ID
        * Keystone url - OS_AUTH_URL
        * Gohan service name in keystone - GOHAN_SERVICE_NAME
        * Gohan region in keystone - GOHAN_REGION
        * Gohan schema URL - GOHAN_SCHEMA_URL
        * Should Client cache schemas (default - true) - GOHAN_CACHE_SCHEMAS
        * Cache expiration time (in format 1h20m10s - default 5m) - GOHAN_CACHE_TIMEOUT
        * Cache path (default - /tmp/.cached-gohan-schemas) - GOHAN_CACHE_PATH
    Additional options for Keystone v3 only:
        * Keystone domain name or domain id - OS_DOMAIN_NAME or OS_DOMAIN_ID
`,
		Action: func(c *cli.Context) {
			opts, err := client.NewOptsFromEnv()
			gohanCLI, err := client.NewGohanClientCLI(opts)
			if err != nil {
				util.ExitFatalf("Error initializing Gohan Client CLI: %v\n", err)
			}

			command := fmt.Sprintf("%s %s", c.Args().Get(0), c.Args().Get(1))
			arguments := c.Args().Tail()
			if len(arguments) > 0 {
				arguments = arguments[1:]
			}
			result, err := gohanCLI.ExecuteCommand(command, arguments)
			if err != nil {
				util.ExitFatalf("%v\n", err)
			}
			if result == "null" {
				result = ""
			}
			fmt.Println(result)
		},
	}
}

func getValidateCommand() cli.Command {
	return cli.Command{
		Name:      "validate",
		ShortName: "v",
		Usage:     "Validate document",
		Description: `
Validate document against schema.
It's especially useful to validate schema files against gohan meta-schema.`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "schema, s", Value: "etc/schema/gohan.json", Usage: "Schema path"},
			cli.StringFlag{Name: "document, d", Value: "etc/apps/example.json", Usage: "Document path"},
		},
		Action: func(c *cli.Context) {
			schemaPath := c.String("schema")
			documentPath := c.String("document")

			manager := schema.GetManager()
			err := manager.LoadSchemaFromFile(schemaPath)
			if err != nil {
				util.ExitFatal("Failed to parse schema:", err)
			}

			err = manager.LoadSchemaFromFile(documentPath)
			if err == nil {
				fmt.Println("Schema is valid")
			} else {
				util.ExitFatalf("Schema is not valid, see errors below:\n%s\n", err)
			}
		},
	}
}

func getInitDbCommand() cli.Command {
	return cli.Command{
		Name:      "init-db",
		ShortName: "idb",
		Usage:     "Initialize DB backend with given schema file",
		Description: `
Initialize empty database with given schema.

Setting meta-schema option will additionaly populate meta-schema table with schema resources.
Useful for development purposes.`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "database-type, t", Value: "sqlite3", Usage: "Backend datebase type"},
			cli.StringFlag{Name: "database, d", Value: "gohan.db", Usage: "DB connection string"},
			cli.StringFlag{Name: "schema, s", Value: "etc/schema/gohan.json", Usage: "Schema definition"},
			cli.BoolFlag{Name: "drop-on-create", Usage: "If true, old database will be dropped"},
			cli.BoolFlag{Name: "cascade", Usage: "If true, FOREIGN KEYS in database will be created with ON DELETE CASCADE"},
			cli.StringFlag{Name: "meta-schema, m", Value: "", Usage: "Meta-schema file (optional)"},
		},
		Action: func(c *cli.Context) {
			dbType := c.String("database-type")
			dbConnection := c.String("database")
			schemaFile := c.String("schema")
			metaSchemaFile := c.String("meta-schema")
			dropOnCreate := c.Bool("drop-on-create")
			cascade := c.Bool("cascade")
			manager := schema.GetManager()
			manager.LoadSchemasFromFiles(schemaFile, metaSchemaFile)
			err := db.InitDBWithSchemas(dbType, dbConnection, dropOnCreate, cascade)
			if err != nil {
				util.ExitFatal(err)
			}
			fmt.Println("DB is initialized")
		},
	}
}

func getConvertCommand() cli.Command {
	return cli.Command{
		Name:      "convert",
		ShortName: "conv",
		Usage:     "Convert DB",
		Description: `
Gohan convert can be used to migrate Gohan resources between different types of databases.

Setting meta-schema option will additionaly convert meta-schema table with schema resources.
Useful for development purposes.`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "in-type, it", Value: "", Usage: "Input db type (yaml, json, sqlite3, mysql)"},
			cli.StringFlag{Name: "in, i", Value: "", Usage: "Input db connection spec (or filename)"},
			cli.StringFlag{Name: "out-type, ot", Value: "", Usage: "Output db type (yaml, json, sqlite3, mysql)"},
			cli.StringFlag{Name: "out, o", Value: "", Usage: "Output db connection spec (or filename)"},
			cli.StringFlag{Name: "schema, s", Value: "", Usage: "Schema file"},
			cli.StringFlag{Name: "meta-schema, m", Value: "", Usage: "Meta-schema file (optional)"},
		},
		Action: func(c *cli.Context) {
			inType, in := c.String("in-type"), c.String("in")
			if inType == "" || in == "" {
				util.ExitFatal("Need to provide input database specification")
			}
			outType, out := c.String("out-type"), c.String("out")
			if outType == "" || out == "" {
				util.ExitFatal("Need to provide output database specification")
			}

			schemaFile := c.String("schema")
			if schemaFile == "" {
				util.ExitFatal("Need to provide schema file")
			}
			metaSchemaFile := c.String("meta-schema")

			schemaManager := schema.GetManager()
			err := schemaManager.LoadSchemasFromFiles(schemaFile, metaSchemaFile)
			if err != nil {
				util.ExitFatal("Error loading schema:", err)
			}

			inDB, err := db.ConnectDB(inType, in)
			if err != nil {
				util.ExitFatal(err)
			}
			outDB, err := db.ConnectDB(outType, out)
			if err != nil {
				util.ExitFatal(err)
			}

			err = db.CopyDBResources(inDB, outDB)
			if err != nil {
				util.ExitFatal(err)
			}

			fmt.Println("Conversion complete")
		},
	}
}

func getServerCommand() cli.Command {
	return cli.Command{
		Name:        "server",
		ShortName:   "srv",
		Usage:       "Run API Server",
		Description: "Run Gohan API server",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: "etc/gohan.yaml", Usage: "Server config File"},
		},
		Action: func(c *cli.Context) {
			configFile := c.String("config-file")
			server.RunServer(configFile)
		},
	}
}

func getTestExtesionsCommand() cli.Command {
	return cli.Command{
		Name:      "test_extensions",
		ShortName: "test_ex",
		Usage:     "Run extension tests",
		Description: `
Run extensions tests in a gohan-server-like environment.

Test files and directories can be supplied as arguments. See Gohan
documentation for detail information about writing tests.`,
		Action: framework.RunTests,
	}
}

func getMigrateCommand() cli.Command {
	return cli.Command{
		Name:        "migrate",
		ShortName:   "mig",
		Usage:       "Generate goose migration script",
		Description: `Generates goose migraion script`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "name, n", Value: "init_schema", Usage: "name of migrate"},
			cli.StringFlag{Name: "schema, s", Value: "", Usage: "Schema definition"},
			cli.StringFlag{Name: "path, p", Value: "etc/db/migrations", Usage: "Migrate path"},
			cli.BoolFlag{Name: "cascade", Usage: "If true, FOREIGN KEYS in database will be created with ON DELETE CASCADE"},
		},
		Action: func(c *cli.Context) {
			schemaFile := c.String("schema")
			cascade := c.Bool("cascade")
			manager := schema.GetManager()
			manager.LoadSchemasFromFiles(schemaFile)
			name := c.String("name")
			now := time.Now()
			version := fmt.Sprintf("%s_%s.sql", now.Format("20060102150405"), name)
			path := filepath.Join(c.String("path"), version)
			var sqlString = bytes.NewBuffer(make([]byte, 0, 100))
			fmt.Printf("Generating goose migration file to %s ...\n", path)
			sqlDB := sql.NewDB()
			schemas := manager.Schemas()
			sqlString.WriteString("\n")
			sqlString.WriteString("-- +goose Up\n")
			sqlString.WriteString("-- SQL in section 'Up' is executed when this migration is applied\n")
			for _, s := range schemas {
				sqlString.WriteString(sqlDB.GenTableDef(s, cascade))
				sqlString.WriteString("\n")
			}
			sqlString.WriteString("\n")
			sqlString.WriteString("-- +goose Down\n")
			sqlString.WriteString("-- SQL section 'Down' is executed when this migration is rolled back\n")
			for _, s := range schemas {
				sqlString.WriteString(fmt.Sprintf("drop table %s;", s.GetDbTableName()))
				sqlString.WriteString("\n\n")
			}
			err := ioutil.WriteFile(path, sqlString.Bytes(), os.ModePerm)
			if err != nil {
				fmt.Println(err)
			}
		},
	}
}
