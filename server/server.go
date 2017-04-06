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

package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/braintree/manners"
	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/migration"
	"github.com/cloudwan/gohan/job"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"
	sync_util "github.com/cloudwan/gohan/sync/util"
	"github.com/cloudwan/gohan/util"
	"github.com/drone/routes"
	"github.com/go-martini/martini"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/martini-contrib/staticbin"
)

type tlsConfig struct {
	CertFile string
	KeyFile  string
}

//Server is a struct for GohanAPIServer
type Server struct {
	address          string
	tls              *tlsConfig
	documentRoot     string
	db               db.DB
	sync             sync.Sync
	running          bool
	martini          *martini.ClassicMartini
	extensions       []string
	keystoneIdentity middleware.IdentityService
	queue            *job.Queue
}

func (server *Server) mapRoutes() {
	config := util.GetConfig()
	schemaManager := schema.GetManager()
	MapNamespacesRoutes(server.martini)
	MapRouteBySchemas(server, server.db)

	tx, err := server.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Close()
	coreSchema, _ := schemaManager.Schema("schema")
	if coreSchema == nil {
		log.Fatal("Gohan core schema not found")
		return
	}

	policySchema, _ := schemaManager.Schema("policy")
	policyList, _, err := tx.List(policySchema, nil, nil)
	if err != nil {
		log.Info(err.Error())
	}
	schemaManager.LoadPolicies(policyList)

	extensionSchema, _ := schemaManager.Schema("extension")
	extensionList, _, err := tx.List(extensionSchema, nil, nil)
	if err != nil {
		log.Info(err.Error())
	}
	schemaManager.LoadExtensions(extensionList)

	namespaceSchema, _ := schemaManager.Schema("namespace")
	if namespaceSchema == nil {
		log.Error("No gohan schema. Disabling schema editing mode")
		return
	}
	namespaceList, _, err := tx.List(namespaceSchema, nil, nil)
	if err != nil {
		log.Info(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		log.Info(err.Error())
	}
	schemaManager.LoadNamespaces(namespaceList)

	if config.GetBool("keystone/fake", false) {
		middleware.FakeKeystone(server.martini)
	}
}

func (server *Server) addOptionsRoute() {
	server.martini.AddRoute("OPTIONS", ".*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (server *Server) resetRouter() {
	router := martini.NewRouter()
	server.martini.Router = router
	server.martini.MapTo(router, (*martini.Routes)(nil))
	server.martini.Action(router.Handle)
	server.addOptionsRoute()
}

func (server *Server) initDB() error {
	return db.InitDBWithSchemas(server.getDatabaseConfig())
}

func (server *Server) connectDB() error {
	if err := migration.Init(); err != nil {
		return err
	}
	config := util.GetConfig()
	dbConn, err := db.CreateFromConfig(config)
	if server.sync == nil {
		server.db = dbConn
	} else {
		server.db = &DbSyncWrapper{dbConn}
	}
	return err
}

func (server *Server) getDatabaseConfig() (string, string, bool, bool, bool) {
	config := util.GetConfig()
	databaseType := config.GetString("database/type", "sqlite3")
	if databaseType == "json" || databaseType == "yaml" {
		log.Fatal("json or yaml isn't supported as main db backend")
	}
	databaseConnection := config.GetString("database/connection", "")
	if databaseConnection == "" {
		log.Fatal("no database connection specified in the configuraion file.")
	}
	databaseDropOnCreate := config.GetBool("database/drop_on_create", false)
	databaseCascade := config.GetBool("database/cascade_delete", false)
	databaseAutoMigrate := config.GetBool("database/auto_migrate", true)
	return databaseType, databaseConnection, databaseDropOnCreate, databaseCascade, databaseAutoMigrate
}

//NewServer returns new GohanAPIServer
func NewServer(configFile string) (*Server, error) {
	manager := schema.GetManager()
	config := util.GetConfig()
	err := config.ReadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("Config load error: %s", err)
	}
	err = os.Chdir(path.Dir(configFile))
	if err != nil {
		return nil, fmt.Errorf("Chdir error: %s", err)
	}
	err = l.SetUpLogging(config)
	if err != nil {
		return nil, fmt.Errorf("Logging setup error: %s", err)
	}
	log.Info("logging initialized")

	server := &Server{}

	m := martini.Classic()
	m.Handlers()
	m.Use(middleware.Logging())
	m.Use(martini.Recovery())
	m.Use(middleware.JSONURLs())
	m.Use(middleware.WithContext())

	server.martini = m

	port := os.Getenv("PORT")

	if port == "" {
		port = "9091"
	}

	setupEditor(server)

	server.extensions = config.GetStringList("extension/use", []string{
		"javascript",
		"gohanscript",
		"go",
	})
	schema.DefaultExtension = config.GetString("extension/default", "javascript")

	manager.TimeLimit = time.Duration(config.GetInt("extension/timelimit", 30)) * time.Second

	if config.GetList("extension/timelimits", nil) != nil {
		timeLimitList := config.GetList("extension/timelimits", nil)
		for _, timeLimit := range timeLimitList {
			cfgRaw := timeLimit.(map[string]interface{})
			cfgPath := cfgRaw["path"].(string)
			cfgEvent := cfgRaw["event"].(string)
			cfgTimeDuration := cfgRaw["timelimit"].(int)

			manager.TimeLimits = append(manager.TimeLimits, &schema.PathEventTimeLimit{
				PathRegex:    regexp.MustCompile(cfgPath),
				EventRegex:   regexp.MustCompile(cfgEvent),
				TimeDuration: time.Second * time.Duration(cfgTimeDuration),
			})
		}
	}

	server.address = config.GetString("address", ":"+port)
	if config.GetBool("tls/enabled", false) {
		log.Info("TLS enabled")
		server.tls = &tlsConfig{
			KeyFile:  config.GetString("tls/key_file", "./etc/key.pem"),
			CertFile: config.GetString("tls/cert_file", "./etc/cert.pem"),
		}
	}

	server.sync, err = sync_util.CreateFromConfig(config)
	if err != nil {
		log.Error("Failed to create sync, err: %s", err)
		return nil, err
	}

	if dbErr := server.connectDB(); dbErr != nil {
		log.Fatalf("Error while connecting to DB: %s", dbErr)
	}

	schemaFiles := config.GetStringList("schemas", nil)
	if schemaFiles == nil {
		log.Fatal("No schema specified in configuraion")
	} else {
		err = manager.LoadSchemasFromFiles(schemaFiles...)
		if err != nil {
			return nil, fmt.Errorf("invalid schema: %s", err)
		}
	}

	if !config.GetBool("database/no_init", false) {
		server.initDB()
	}

	if config.GetList("database/initial_data", nil) != nil {
		initialDataList := config.GetList("database/initial_data", nil)
		for _, initialData := range initialDataList {
			initialDataConfig := initialData.(map[string]interface{})
			inType := initialDataConfig["type"].(string)
			inConnection := initialDataConfig["connection"].(string)
			log.Info("Importing data from %s ...", inConnection)
			inDB, err := db.ConnectDB(inType, inConnection, db.DefaultMaxOpenConn)
			if err != nil {
				log.Fatal(err)
			}
			db.CopyDBResources(inDB, server.db, false)
		}
	}

	m.Map(middleware.NewNobodyResourceService(manager.NobodyResourcePaths()))

	if config.GetBool("keystone/use_keystone", false) {
		server.keystoneIdentity, err = middleware.CreateIdentityServiceFromConfig(config)
		m.MapTo(server.keystoneIdentity, (*middleware.IdentityService)(nil))
		m.Use(middleware.Authentication())
	} else {
		m.MapTo(&middleware.NoIdentityService{}, (*middleware.IdentityService)(nil))
		m.Map(schema.NewAuthorization("admin", "admin", "admin_token", []string{"admin"}, nil))
	}

	if err != nil {
		return nil, fmt.Errorf("invalid base dir: %s", err)
	}

	server.addOptionsRoute()
	cors := config.GetString("cors", "")
	if cors != "" {
		log.Info("Enabling CORS for %s", cors)
		if cors == "*" {
			log.Warning("cors for * have security issue")
		}
		server.martini.Use(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Add("Access-Control-Allow-Origin", cors)
			rw.Header().Add("Access-Control-Allow-Headers", "X-Auth-Token, Content-Type")
			rw.Header().Add("Access-Control-Expose-Headers", "X-Total-Count")
			rw.Header().Add("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE")
		})
	}

	documentRoot := config.GetString("document_root", "embed")
	if config.GetBool("webui_config/enabled", false) {
		m.Use(func(res http.ResponseWriter, req *http.Request, c martini.Context) {
			if req.URL.Path != "/webui/config.json" {
				c.Next()
				return
			}
			address := config.GetString("webui_config/address", server.address)
			if address[0] == ':' {
				address = "__HOST__" + address
			}
			baseURL := "http://" + address
			authURL := "http://" + address + "/v2.0"
			if config.GetBool("webui_config/tls", config.GetBool("tls/enabled", false)) {
				baseURL = "https://" + address
				authURL = "https://" + address + "/v2.0"
			}
			authURL = config.GetString("webui_config/auth_url", authURL)
			webUIConfig := map[string]interface{}{
				"authUrl": authURL,
				"gohan": map[string]interface{}{
					"schema": "/gohan/v0.1/schemas",
					"url":    baseURL,
				},
				"routes": []interface{}{
					map[string]interface{}{
						"path":      "",
						"viewClass": "topView",
						"name":      "top_view",
					},
				},
				"errorMessages": map[string]interface{}{
					"tokenExpire": "The token is expired. Please re-login.",
				},
				"addingRelationDialog": []interface{}{
					"Pet",
				},
				"pageLimit":           25,
				"loginRequestTimeout": 30000,
				"extendTokenTime":     300000,
			}
			routes.ServeJson(res, webUIConfig)
		})
	}
	if documentRoot == "embed" {
		m.Use(staticbin.Static("public", util.Asset, staticbin.Options{
			SkipLogging: true,
		}))
	} else {
		log.Info("Static file serving from %s", documentRoot)
		documentRootABS, err := filepath.Abs(documentRoot)
		if err != nil {
			return nil, err
		}
		server.martini.Use(martini.Static(documentRootABS, martini.StaticOptions{
			SkipLogging: true,
		}))
	}
	server.mapRoutes()

	maxWorkerCount := config.GetInt("workers", 100)
	server.queue = job.NewQueue(uint(maxWorkerCount))

	return server, nil
}

//Start starts GohanAPIServer
func (server *Server) Start() (err error) {
	listeners, err := listener.ListenAll()
	var l net.Listener
	if err != nil || len(listeners) == 0 {
		l, err = net.Listen("tcp", server.address)
		if err != nil {
			return err
		}
	} else {
		l = listeners[0]
	}
	if server.tls != nil {
		config := &tls.Config{ClientAuth: tls.VerifyClientCertIfGiven}
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(server.tls.CertFile, server.tls.KeyFile)
		if err != nil {
			return err
		}
		l = tls.NewListener(l, config)
	}
	return manners.Serve(l, server.martini)
}

//Router returns http handler
func (server *Server) Router() http.Handler {
	return server.martini
}

//Stop stops GohanAPIServer
func (server *Server) Stop() {
	server.running = false
	if server.sync != nil {
		stopSyncProcess(server)
		stopStateWatchProcess(server)
		stopSyncWatchProcess(server)
	}
	stopAMQPProcess(server)
	stopSNMPProcess(server)
	stopCRONProcess(server)
	manners.Close()
	server.queue.Stop()
}

//Queue returns servers build-in queue
func (server *Server) Queue() *job.Queue {
	return server.queue
}

//RunServer runs gohan api server
func RunServer(configFile string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	server, err := NewServer(configFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Gohan no jikan desuyo (It's time for dinner!) ")
	log.Info("Starting Gohan Server...")
	address := server.address
	if strings.HasPrefix(address, ":") {
		address = "localhost" + address
	}
	protocol := "http"
	if server.tls != nil {
		protocol = "https"
	}
	log.Info("    API Server %s://%s/", protocol, address)
	log.Info("    Web UI %s://%s/webui/", protocol, address)
	go func() {
		for _ = range c {
			log.Info("Stopping the server...")
			log.Info("Tearing down...")
			log.Info("Stopping server...")
			server.Stop()
		}
	}()
	server.running = true

	if server.sync != nil {
		startSyncProcess(server)
		startStateWatchProcess(server)
		startSyncWatchProcess(server)
	}
	startAMQPProcess(server)
	startSNMPProcess(server)
	startCRONProcess(server)
	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func startAMQPNotificationProcess(server *Server) {

}
