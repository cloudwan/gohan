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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/lestrrat/go-server-starter"

	"github.com/cloudwan/gohan/cli/client"
	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/extension/framework"
	"github.com/cloudwan/gohan/server"
	"github.com/codegangsta/cli"

	"github.com/cloudwan/gohan/extension/gohanscript"
	//Import gohan script lib

	"github.com/cloudwan/gohan/db/migration"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/extension"
	_ "github.com/cloudwan/gohan/extension/gohanscript/autogen"
	"github.com/cloudwan/gohan/extension/otto"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/server/middleware"
	sync_util "github.com/cloudwan/gohan/sync/util"
)

var log = l.NewLogger()

const defaultConfigFile = "gohan.yaml"

//Run execute main command
func Run(name, usage, version string) {
	app := cli.NewApp()
	app.Name = "gohan"
	app.Usage = "Gohan"
	app.Version = version
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
		getTemplateCommand(),
		getRunCommand(),
		getTestCommand(),
		getOpenAPICommand(),
		getMarkdownCommand(),
		getDotCommand(),
		getGraceServerCommand(),
		getGenerateCommand(),
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
                --fields [field1,field2] - specifies which fields should be visible (default all)
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
        * In which format results should be shown, see --output-format - GOHAN_OUTPUT_FORMAT
        * How much debug info Gohan Client should show, see --verbosity - GOHAN_VERBOSITY
        * Which columns should be visible in results, see --fields - GOHAN_FIELDS
    Additional options for Keystone v3 only:
        * Keystone domain name or domain id - OS_DOMAIN_NAME or OS_DOMAIN_ID
`,
		Action: func(c *cli.Context) {
			opts, err := client.NewOptsFromEnv()
			if err != nil {
				util.ExitFatal(err)
			}

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
				util.ExitFatal(err)
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
			cli.StringSliceFlag{Name: "document, d", Usage: "Document path"},
		},
		Action: func(c *cli.Context) {
			schemaPath := c.String("schema")
			documentPaths := c.StringSlice("document")
			if len(documentPaths) == 0 {
				util.ExitFatalf("At least one document should be specified for validation\n")
			}

			manager := schema.GetManager()
			err := manager.LoadSchemaFromFile(schemaPath)
			if err != nil {
				util.ExitFatal("Failed to parse schema:", err)
			}

			for _, documentPath := range documentPaths {
				err = manager.LoadSchemaFromFile(documentPath)
				if err != nil {
					util.ExitFatalf("Schema is not valid, see errors below:\n%s\n", err)
				}
			}
			fmt.Println("Schema is valid")
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

Setting meta-schema option will additionally populate meta-schema table with schema resources.
Useful for development purposes.`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "database-type, t", Value: "sqlite3", Usage: "Backend datebase type"},
			cli.StringFlag{Name: "database, d", Value: "gohan.db", Usage: "DB connection string"},
			cli.StringFlag{Name: "schema, s", Value: "embed://etc/schema/gohan.json", Usage: "Schema definition"},
			cli.BoolFlag{Name: "drop-on-create", Usage: "If true, old database will be dropped"},
			cli.BoolFlag{Name: "cascade", Usage: "If true, FOREIGN KEYS in database will be created with ON DELETE CASCADE"},
			cli.StringFlag{Name: "meta-schema, m", Value: "", Usage: "Meta-schema file (optional)"},
			cli.StringFlag{Name: "multiple-schemas", Value: "", Usage: "Multiple schema files separated by semicolon (;)"},
		},
		Action: func(c *cli.Context) {
			dbType := c.String("database-type")
			dbConnection := c.String("database")
			schemaFile := c.String("schema")
			metaSchemaFile := c.String("meta-schema")
			multipleSchemaFiles := c.String("multiple-schemas")
			dropOnCreate := c.Bool("drop-on-create")
			cascade := c.Bool("cascade")
			manager := schema.GetManager()
			manager.LoadSchemasFromFiles(schemaFile, metaSchemaFile)
			manager.OrderedLoadSchemasFromFiles(strings.Split(multipleSchemaFiles, ";"))
			err := db.InitDBWithSchemas(dbType, dbConnection, dropOnCreate, cascade, false)
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

Setting meta-schema option will additionally convert meta-schema table with schema resources.
Useful for development purposes.`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "in-type, it", Value: "", Usage: "Input db type (yaml, json, sqlite3, mysql)"},
			cli.StringFlag{Name: "in, i", Value: "", Usage: "Input db connection spec (or filename)"},
			cli.StringFlag{Name: "out-type, ot", Value: "", Usage: "Output db type (yaml, json, sqlite3, mysql)"},
			cli.StringFlag{Name: "out, o", Value: "", Usage: "Output db connection spec (or filename)"},
			cli.StringFlag{Name: "schema, s", Value: "", Usage: "Schema file"},
			cli.StringFlag{Name: "meta-schema, m", Value: "embed://etc/schema/gohan.json", Usage: "Meta-schema file (optional)"},
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

			inDB, err := db.ConnectDB(inType, in, db.DefaultMaxOpenConn)
			if err != nil {
				util.ExitFatal(err)
			}
			outDB, err := db.ConnectDB(outType, out, db.DefaultMaxOpenConn)
			if err != nil {
				util.ExitFatal(err)
			}

			err = db.CopyDBResources(inDB, outDB, true)
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
			cli.StringFlag{Name: "config-file", Value: defaultConfigFile, Usage: "Server config File"},
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
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "verbose, v", Usage: "Print logs for passing tests"},
			cli.StringFlag{Name: "config-file,c", Value: "", Usage: "Config file path"},
			cli.StringFlag{Name: "run-test,r", Value: "", Usage: "Run only tests matching specified regex"},
			cli.IntFlag{Name: "parallel, p", Value: runtime.NumCPU(), Usage: "Allow parallel execution of test functions"},
		},
		Action: framework.TestExtensions,
	}
}

func getMigrateSubcommand(subcmd, usage string) cli.Command {
	return cli.Command{
		Name:  subcmd,
		Usage: usage,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: defaultConfigFile, Usage: "Server config File"},
		},
		Action: func(context *cli.Context) {
			configFile := context.String("config-file")
			if migration.LoadConfig(configFile) != nil {
				return
			}
			migration.Run(subcmd, context.Args())
		},
	}
}

func getMigrateSubcommandWithPostMigrateEvent(subcmd, usage string) cli.Command {
	const (
		POST_MIGRATION_EVENT_TIMEOUT_FLAG = "post-migration-event-timeout"
		CONFIG_FILE_FLAG                  = "config-file"
		EMIT_POST_MIGRATION_EVENT_FLAG    = "emit-post-migration-event"
		POST_MIGRATION_EVENT              = "post-migration"
	)
	return cli.Command{
		Name:  subcmd,
		Usage: usage,
		Flags: []cli.Flag{
			cli.StringFlag{Name: CONFIG_FILE_FLAG, Value: defaultConfigFile, Usage: "Server config File"},
			cli.BoolFlag{Name: EMIT_POST_MIGRATION_EVENT_FLAG, Usage: "Enable if post-migration event should be emited to modified schema extensions"},
			cli.DurationFlag{Name: POST_MIGRATION_EVENT_TIMEOUT_FLAG, Value: time.Second * 30, Usage: "Maximum duration of post-migration event"},
		},
		Action: func(context *cli.Context) {
			configFile := context.String(CONFIG_FILE_FLAG)
			if migration.LoadConfig(configFile) != nil {
				return
			}
			config := util.GetConfig()

			migration.Run(subcmd, context.Args())

			emitEvent := context.Bool(EMIT_POST_MIGRATION_EVENT_FLAG)
			if !emitEvent {
				return
			}

			modifiedSchemas := migration.GetModifiedSchemas()

			if len(modifiedSchemas) == 0 {
				log.Info("No modified schemas, skipping post-migration event")
				return
			}

			log.Debug("Modified schemas: %s", modifiedSchemas)

			schemaFiles := config.GetStringList("schemas", nil)
			if schemaFiles == nil {
				log.Fatal("No schema specified in configuraion")
			}

			manager := schema.GetManager()
			if err := manager.LoadSchemasFromFiles(schemaFiles...); err != nil {
				log.Fatal(err)
			}

			dbConn, err := db.CreateFromConfig(config)
			if err != nil {
				log.Fatal(err)
			}

			sync, err := sync_util.CreateFromConfig(config)
			if err != nil {
				log.Fatal(err)
			}

			identity, err := middleware.CreateIdentityServiceFromConfig(config)
			if err != nil {
				log.Fatal(err)
			}

			environmentManager := extension.GetManager()

			for _, s := range manager.Schemas() {
				if !util.ContainsString(modifiedSchemas, s.ID) {
					continue
				}
				pluralURL := s.GetPluralURL()

				if _, ok := environmentManager.GetEnvironment(s.ID); !ok {
					env := otto.NewEnvironment("post-migration-env", dbConn, identity, sync)
					eventTimeout := context.Duration(POST_MIGRATION_EVENT_TIMEOUT_FLAG)
					env.SetEventTimeLimit(POST_MIGRATION_EVENT, eventTimeout)
					env.LoadExtensionsForPath(manager.Extensions, manager.TimeLimit, manager.TimeLimits, pluralURL)
					log.Info("Loading environment for %s schema with URL: %s", s.ID, pluralURL)
					if err != nil {
						log.Fatal(fmt.Sprintf("[%s] %v", pluralURL, err))
					}
					environmentManager.RegisterEnvironment(s.ID, env)
				}

				env, _ := environmentManager.GetEnvironment(s.ID)
				eventContext := map[string]interface{}{}
				eventContext["schema"] = s
				eventContext["sync"] = sync
				eventContext["db"] = dbConn
				eventContext["identity_service"] = identity
				err := env.HandleEvent(POST_MIGRATION_EVENT, eventContext)
				if err != nil {
					log.Fatalf("Failed to handle event post-migration, err: %s", err)
				}
			}
		},
	}

}

func getMigrateCommand() cli.Command {
	return cli.Command{
		Name:      "migrate",
		ShortName: "mig",
		Usage:     "Manage migrations",
		Subcommands: []cli.Command{
			getMigrateSubcommandWithPostMigrateEvent("up", "Migrate to the most recent version"),
			getMigrateSubcommandWithPostMigrateEvent("up-by-one", "Migrate one version up"),
			getMigrateSubcommandWithPostMigrateEvent("up-to", "Migrate up to specific version"),
			getMigrateSubcommand("create", "Create a template for a new migration"),
			getCreateInitialMigrationCommand(),
			getMigrateSubcommand("down", "Migrate to the oldest version"),
			getMigrateSubcommand("down-to", "Migrate to specific version"),
			getMigrateSubcommand("redo", "Migrate one version back"),
			getMigrateSubcommand("status", "Display migration status"),
			getMigrateSubcommand("version", "Display migration version"),
		},
		Action: func(c *cli.Context) {
			migration.Help()
		},
	}
}

func getCreateInitialMigrationCommand() cli.Command {
	return cli.Command{
		Name:        "initial",
		ShortName:   "init",
		Usage:       "Generate initial goose migration script from schema",
		Description: `Generates initial goose migraion script from schema`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "name, n", Value: "init_schema", Usage: "name of migrate"},
			cli.StringFlag{Name: "schema, s", Value: "", Usage: "Schema definition"},
			cli.StringFlag{Name: "config-file, c", Value: defaultConfigFile, Usage: "Config file"},
			cli.StringFlag{Name: "path, p", Value: "etc/db/migrations", Usage: "Migrate path"},
			cli.BoolFlag{Name: "cascade", Usage: "If true, FOREIGN KEYS in database will be created with ON DELETE CASCADE"},
		},
		Action: func(c *cli.Context) {
			schemaFile := c.String("schema")
			cascade := c.Bool("cascade")
			manager := schema.GetManager()
			configFile := c.String("config-file")
			if configFile != "" {
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
			}
			if schemaFile != "" {
				manager.LoadSchemasFromFiles(schemaFile)
			}
			name := c.String("name")
			now := time.Now()
			version := fmt.Sprintf("%s_%s.sql", now.Format("20060102150405"), name)
			path := filepath.Join(c.String("path"), version)
			var sqlString = bytes.NewBuffer(make([]byte, 0, 100))
			fmt.Printf("Generating goose migration file to %s ...\n", path)
			sqlDB := sql.NewDB()
			schemas := manager.OrderedSchemas()
			sqlString.WriteString("\n")
			sqlString.WriteString("-- +goose Up\n")
			sqlString.WriteString("-- SQL in section 'Up' is executed when this migration is applied\n")
			for _, s := range schemas {
				if s.IsAbstract() {
					continue
				}
				if s.Metadata["type"] == "metaschema" {
					continue
				}
				createSql, indices := sqlDB.GenTableDef(s, cascade)
				sqlString.WriteString(createSql + "\n")
				for _, indexSql := range indices {
					sqlString.WriteString(indexSql + "\n")
				}
			}
			sqlString.WriteString("\n")
			sqlString.WriteString("-- +goose Down\n")
			sqlString.WriteString("-- SQL section 'Down' is executed when this migration is rolled back\n")
			for _, s := range schemas {
				if s.IsAbstract() {
					continue
				}
				if s.Metadata["type"] == "metaschema" {
					continue
				}
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

func getRunCommand() cli.Command {
	return cli.Command{
		Name:      "run",
		ShortName: "run",
		Usage:     "Run Gohan script Code",
		Description: `
Run gohan script code.`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file,c", Value: defaultConfigFile, Usage: "config file path"},
			cli.StringFlag{Name: "args,a", Value: "", Usage: "arguments"},
		},
		Action: func(c *cli.Context) {
			src := c.Args()[0]
			vm := gohanscript.NewVM()

			args := []interface{}{}
			flags := map[string]interface{}{}
			for _, arg := range c.Args()[1:] {
				if strings.Contains(arg, "=") {
					kv := strings.Split(arg, "=")
					flags[kv[0]] = kv[1]
				} else {
					args = append(args, arg)
				}
			}
			vm.Context.Set("args", args)
			vm.Context.Set("flags", flags)
			configFile := c.String("config-file")
			loadConfig(configFile)
			_, err := vm.RunFile(src)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
				return
			}
		},
	}
}

func getTestCommand() cli.Command {
	return cli.Command{
		Name:      "test",
		ShortName: "test",
		Usage:     "Run Gohan script Test",
		Description: `
Run gohan script yaml code.`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file,c", Value: defaultConfigFile, Usage: "config file path"},
		},
		Action: func(c *cli.Context) {
			dir := c.Args()[0]
			configFile := c.String("config-file")
			loadConfig(configFile)
			gohanscript.RunTests(dir)
		},
	}
}

func loadConfig(configFile string) {
	if configFile == "" {
		return
	}
	config := util.GetConfig()
	err := config.ReadConfig(configFile)
	if err != nil {
		if configFile != defaultConfigFile {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}
	err = l.SetUpLogging(config)
	if err != nil {
		fmt.Printf("Logging setup error: %s\n", err)
		os.Exit(1)
		return
	}
	log.Info("logging initialized")
}

type options struct {
	OptArgs                []string
	OptCommand             string
	OptDir                 string   `long:"dir" arg:"path" description:"working directory, start_server do chdir to before exec (optional)"`
	OptInterval            int      `long:"interval" arg:"seconds" description:"minimum interval (in seconds) to respawn the server program (default: 1)"`
	OptPorts               []string `long:"port" arg:"(port|host:port)" description:"TCP port to listen to (if omitted, will not bind to any ports)"`
	OptPaths               []string `long:"path" arg:"path" description:"path at where to listen using unix socket (optional)"`
	OptSignalOnHUP         string   `long:"signal-on-hup" arg:"Signal" description:"name of the signal to be sent to the server process when start_server\nreceives a SIGHUP (default: SIGTERM). If you use this option, be sure to\nalso use '--signal-on-term' below."`
	OptSignalOnTERM        string   `long:"signal-on-term" arg:"Signal" description:"name of the signal to be sent to the server process when start_server\nreceives a SIGTERM (default: SIGTERM)"`
	OptPidFile             string   `long:"pid-file" arg:"filename" description:"if set, writes the process id of the start_server process to the file"`
	OptStatusFile          string   `long:"status-file" arg:"filename" description:"if set, writes the status of the server process(es) to the file"`
	OptEnvdir              string   `long:"envdir" arg:"Envdir" description:"directory that contains environment variables to the server processes.\nIt is intended for use with \"envdir\" in \"daemontools\". This can be\noverwritten by environment variable \"ENVDIR\"."`
	OptEnableAutoRestart   bool     `long:"enable-auto-restart" description:"enables automatic restart by time. This can be overwritten by\nenvironment variable \"ENABLE_AUTO_RESTART\"." note:"unimplemented"`
	OptAutoRestartInterval int      `long:"auto-restart-interval" arg:"seconds" description:"automatic restart interval (default 360). It is used with\n\"--enable-auto-restart\" option. This can be overwritten by environment\nvariable \"AUTO_RESTART_INTERVAL\"." note:"unimplemented"`
	OptKillOldDelay        int      `long:"kill-old-delay" arg:"seconds" description:"time to suspend to send a signal to the old worker. The default value is\n5 when \"--enable-auto-restart\" is set, 0 otherwise. This can be\noverwritten by environment variable \"KILL_OLD_DELAY\"."`
	OptRestart             bool     `long:"restart" description:"this is a wrapper command that reads the pid of the start_server process\nfrom --pid-file, sends SIGHUP to the process and waits until the\nserver(s) of the older generation(s) die by monitoring the contents of\nthe --status-file" note:"unimplemented"`
	OptHelp                bool     `long:"help" description:"prints this help"`
	OptVersion             bool     `long:"version" description:"prints the version number"`
}

func (o options) Args() []string          { return o.OptArgs }
func (o options) Command() string         { return o.OptCommand }
func (o options) Dir() string             { return o.OptDir }
func (o options) Interval() time.Duration { return time.Duration(o.OptInterval) * time.Second }
func (o options) PidFile() string         { return o.OptPidFile }
func (o options) Ports() []string         { return o.OptPorts }
func (o options) Paths() []string         { return o.OptPaths }
func (o options) SignalOnHUP() os.Signal  { return starter.SigFromName(o.OptSignalOnHUP) }
func (o options) SignalOnTERM() os.Signal { return starter.SigFromName(o.OptSignalOnTERM) }
func (o options) StatusFile() string      { return o.OptStatusFile }

func getGraceServerCommand() cli.Command {
	return cli.Command{
		Name:        "glace-server",
		ShortName:   "gsrv",
		Usage:       "Run API Server with graceful restart support",
		Description: "Run Gohan API server with graceful restart support",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "config-file", Value: defaultConfigFile, Usage: "Server config File"},
		},
		Action: func(c *cli.Context) {
			configFile := c.String("config-file")
			loadConfig(configFile)
			opts := &options{OptInterval: -1}
			opts.OptCommand = os.Args[0]
			config := util.GetConfig()
			opts.OptPorts = []string{config.GetString("address", ":9091")}
			opts.OptArgs = []string{"server", "--config-file", configFile}
			s, err := starter.NewStarter(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				return
			}
			s.Run()
		},
	}
}

func getGenerateCommand() cli.Command {
	return cli.Command{
		Name:      "generate",
		ShortName: "gen",
		Usage:     "Generate ServerSide Code",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "template, t", Value: "embed://etc/templates/server.tmpl", Usage: "Application template path"},
			cli.StringFlag{Name: "config-file, c", Value: "./gohan.yaml", Usage: "Gohan config file"},
			cli.StringFlag{Name: "output, o", Value: ".", Usage: "Dir of output"},
			cli.StringFlag{Name: "package, p", Value: "gen", Usage: "Package Name"},
		},
		Action: gohanGenerate,
	}
}

func gohanGenerate(c *cli.Context) {
	path := c.String("output")
	template := c.String("template")
	configFile := c.String("config-file")
	packageName := c.String("package")
	codeDir := filepath.Join(path, packageName)
	etcDir := filepath.Join(path, "etc")
	dbDir := filepath.Join(etcDir, "db")
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
	execCommand(
		fmt.Sprintf(
			"gohan template --config-file %s --template %s | grep -v '^\\s*$' > %s/base_controller.go", configFile, template, packageName))
	execCommand(
		fmt.Sprintf(
			"goimports -w %s/base_controller.go", codeDir))

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
