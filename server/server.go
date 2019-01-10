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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
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
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/initializer"
	"github.com/cloudwan/gohan/db/migration"
	"github.com/cloudwan/gohan/db/transaction"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"
	sync_util "github.com/cloudwan/gohan/sync/util"
	"github.com/cloudwan/gohan/util"
	"github.com/cloudwan/gohan/version"
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

	masterCtx       context.Context
	masterCtxCancel context.CancelFunc
}

func (server *Server) mapRoutes() {
	config := util.GetConfig()
	schemaManager := schema.GetManager()
	mapSchemaRoute(server.martini, schemaManager)
	mapVersionRoute(server.martini, schemaManager)
	MapNamespacesRoutes(server.martini)
	MapRouteBySchemas(server, server.db)

	if txErr := db.WithinTx(server.db, func(tx transaction.Transaction) error {
		ctx := context.Background()
		coreSchema, _ := schemaManager.Schema("schema")
		if coreSchema == nil {
			return fmt.Errorf("Gohan core schema not found")
		}

		policySchema, _ := schemaManager.Schema("policy")
		policyList, _, err := tx.List(ctx, policySchema, nil, nil, nil)
		if err != nil {
			return err
		}

		if err = schemaManager.LoadPolicies(policyList); err != nil {
			return err
		}

		extensionSchema, _ := schemaManager.Schema("extension")
		extensionList, _, err := tx.List(ctx, extensionSchema, nil, nil, nil)
		if err != nil {
			return err
		}
		if err := schemaManager.LoadExtensions(extensionList); err != nil {
			return fmt.Errorf("failed to load extensions: %s", err)
		}

		namespaceSchema, _ := schemaManager.Schema("namespace")
		if namespaceSchema == nil {
			return fmt.Errorf("No gohan schema. Disabling schema editing mode")
		}
		namespaceList, _, err := tx.List(ctx, namespaceSchema, nil, nil, nil)
		if err != nil {
			return err
		}
		schemaManager.LoadNamespaces(namespaceList)

		if config.GetBool("keystone/fake", false) {
			middleware.FakeKeystone(server.martini)
		}

		return nil
	}); txErr != nil {
		log.Fatal(txErr)
	}
}

func (server *Server) addOptionsRoute() {
	server.martini.AddRoute("OPTIONS", ".*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (server *Server) addPprofRoutes() {
	server.martini.Group("/debug/pprof", func(r martini.Router) {
		r.Any("/", pprof.Index)
		r.Any("/cmdline", pprof.Cmdline)
		r.Any("/profile", pprof.Profile)
		r.Any("/symbol", pprof.Symbol)
		r.Any("/trace", pprof.Trace)
		r.Any("/block", pprof.Handler("block").ServeHTTP)
		r.Any("/heap", pprof.Handler("heap").ServeHTTP)
		r.Any("/mutex", pprof.Handler("mutex").ServeHTTP)
		r.Any("/goroutine", pprof.Handler("goroutine").ServeHTTP)
		r.Any("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
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
	return dbutil.InitDBWithSchemas(server.getDatabaseConfig())
}

func (server *Server) connectDB() error {
	if err := migration.Init(); err != nil {
		return err
	}
	config := util.GetConfig()
	dbConn, err := dbutil.CreateFromConfig(config)
	if server.sync == nil {
		server.db = dbConn
	} else {
		server.db = &DbSyncWrapper{dbConn}
	}
	return err
}

func (server *Server) getDatabaseConfig() (string, string, db.InitDBParams) {
	config := util.GetConfig()
	databaseType := config.GetString("database/type", "sqlite3")
	databaseConnection := config.GetString("database/connection", "")
	if databaseConnection == "" {
		log.Fatal("no database connection specified in the configuration file.")
	}
	databaseDropOnCreate := config.GetBool("database/drop_on_create", false)
	databaseCascade := config.GetBool("database/cascade_delete", false)
	databaseAutoMigrate := config.GetBool("database/auto_migrate", true)
	return databaseType, databaseConnection, db.InitDBParams{
		DropOnCreate: databaseDropOnCreate,
		Cascade:      databaseCascade,
		AutoMigrate:  databaseAutoMigrate,
		AllowEmpty:   false,
	}
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
	m.Use(middleware.WithContext())
	m.Use(middleware.Tracing())
	m.Use(middleware.Logging())
	m.Use(middleware.Metrics())
	m.Use(martini.Recovery())
	m.Use(middleware.JSONURLs())

	server.martini = m

	port := os.Getenv("PORT")

	if port == "" {
		port = "9091"
	}

	server.extensions = config.GetStringList("extension/use", []string{
		"goext",
		"javascript",
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
		log.Fatal("No schema specified in configuration")
	} else {
		err = manager.LoadSchemasFromFiles(schemaFiles...)
		if err != nil {
			return nil, fmt.Errorf("invalid schema: %s", err)
		}
	}

	if !config.GetBool("database/no_init", false) {
		server.initDB()
	}

	if err = metrics.SetupMetrics(config); err != nil {
		return nil, err
	}

	if config.GetList("database/initial_data", nil) != nil {
		initialDataList := config.GetList("database/initial_data", nil)
		for _, initialData := range initialDataList {
			initialDataConfig := initialData.(map[string]interface{})
			filePath := initialDataConfig["connection"].(string)
			log.Info("Importing data from %s ...", filePath)
			source, err := initializer.NewInitializer(filePath)
			if err != nil {
				log.Fatal(err)
			}
			dbutil.CopyDBResources(source, server.db, false)
		}
	}

	m.Map(middleware.NewNobodyResourceService(manager.NobodyResourcePaths()))

	if config.GetBool("keystone/use_keystone", false) {
		server.keystoneIdentity, err = middleware.CreateIdentityServiceFromConfig(config)
		m.MapTo(server.keystoneIdentity, (*middleware.IdentityService)(nil))
		m.Use(middleware.Authentication())
	} else {
		m.MapTo(&middleware.NoIdentityService{}, (*middleware.IdentityService)(nil))
		auth := schema.NewAuthorizationBuilder().
			WithTenant(schema.Tenant{ID: "admin", Name: "admin"}).
			WithRoleIDs("admin").
			BuildAdmin()
		m.Map(auth)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid base dir: %s", err)
	}

	if config.GetBool("profiling/enabled", false) {
		server.addPprofRoutes()
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

	return server, nil
}

// Address returns server address.
func (server *Server) Address() string {
	return server.address
}

// GetSync returns server sync.
func (server *Server) GetSync() sync.Sync {
	return server.sync
}

// SetRunning sets server running status.
func (server *Server) SetRunning(running bool) {
	server.running = running
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
	server.masterCtxCancel()
	stopCRONProcess(server)
	manners.Close()
}

//RunServer runs gohan api server
func RunServer(configFile string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	server, err := NewServer(configFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Gohan no jikan desuyo (It's time for dinner!)")
	log.Info("Build version: %s", version.Build.Version)
	log.Info("Build timestamp: %s", version.Build.Timestamp)
	log.Info("Build host: %s", version.Build.Host)
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
		for range c {
			log.Info("Stopping the server...")
			log.Info("Tearing down...")
			log.Info("Stopping server...")
			server.Stop()
		}
	}()
	server.running = true
	server.masterCtx, server.masterCtxCancel = context.WithCancel(context.Background())

	if server.sync != nil {
		stateWatcher := NewStateWatcher(server.sync, server.db, server.keystoneIdentity)
		go stateWatcher.Run(server.masterCtx)

		syncWriter := NewSyncWriter(server.sync, server.db)
		go syncWriter.Run(server.masterCtx)

		syncWatcher := NewSyncWatcherFromServer(server)
		go func(masterCtx context.Context) {
			if err := syncWatcher.Run(masterCtx); err != nil {
				log.Error("An error occurred during SyncWatcher shutdown: %s", err)
			}
		}(server.masterCtx)
	}
	startCRONProcess(server)
	metrics.StartMetricsProcess()
	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}
}
