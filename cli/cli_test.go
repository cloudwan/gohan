// Copyright (C) 2017 NTT Innovation Institute, Inc.
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
	context_pkg "context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/cloudwan/gohan/db/migration"
	"github.com/cloudwan/gohan/extension/framework"
	"github.com/cloudwan/gohan/sync/etcdv3"
	sync_util "github.com/cloudwan/gohan/sync/util"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli"
)

const useEtcdEnv = "USE_ETCD_DURING_MIGRATIONS"

func getContextWithConfig(configFile string, useEtcd bool) *cli.Context {
	if useEtcd {
		os.Setenv(useEtcdEnv, "true")
	}

	configFlag := cli.StringFlag{Name: "config-file", Value: configFile}
	useEtcdFlag := cli.BoolFlag{Name: FlagLockWithETCD, EnvVar: useEtcdEnv}

	set := flag.NewFlagSet("", flag.ContinueOnError)
	configFlag.Apply(set)
	useEtcdFlag.Apply(set)

	return cli.NewContext(nil, set, &cli.Context{})
}

var _ = Describe("CLI", func() {
	const (
		configPath = "../tests/test_etcd.yaml"
		etcdServer = "http://127.0.0.1:2379"
	)

	var (
		waitForThread      sync.WaitGroup
		waitForLocal       sync.WaitGroup
		etcdSync           *etcdv3.Sync
		ctx                context_pkg.Context
		originalWorkingDir string
	)

	BeforeEach(func() {
		ctx = context_pkg.Background()

		waitForThread = sync.WaitGroup{}
		waitForLocal = sync.WaitGroup{}

		waitForThread.Add(1)
		waitForLocal.Add(1)

		var err error
		etcdSync, err = etcdv3.NewSync([]string{etcdServer}, time.Second)
		Expect(err).ToNot(HaveOccurred())

		originalWorkingDir = workingDir()
	})

	AfterEach(func() {
		Expect(os.Unsetenv(useEtcdEnv)).To(Succeed())
		Expect(os.Chdir(originalWorkingDir)).To(Succeed())
	})

	Describe("Post migration subcommand wrapper tests", func() {
		It("Should lock when the flag is set - migrationsSubCommand wrapper", func() {
			lock := func(context_pkg.Context, *cli.Context) {
				waitForThread.Done()
				waitForLocal.Wait()
			}

			context := getContextWithConfig(configPath, true)
			Expect(context.String("config-file")).To(Equal(configPath))

			wrapped := func() {
				withinLockedMigration(lock)(context)
				waitForThread.Done()
			}

			go wrapped()
			waitForThread.Wait()
			waitForThread.Add(1)
			_, err := etcdSync.Lock(ctx, syncMigrationsPath, false)
			waitForLocal.Done()
			waitForThread.Wait()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("failed to lock path %s", syncMigrationsPath)))
		})

		It("Should not lock when the flag is unset - migrationsSubCommand wrapper", func() {
			lock := func() {
				etcdSync.Lock(ctx, syncMigrationsPath, true)
				waitForThread.Done()
				waitForLocal.Wait()
				etcdSync.Unlock(ctx, syncMigrationsPath)
				waitForThread.Done()
			}

			wrapped := withinLockedMigration(func(context_pkg.Context, *cli.Context) {})
			context := getContextWithConfig(configPath, false)
			Expect(context.String("config-file")).To(Equal(configPath))

			go lock()
			waitForThread.Wait()
			waitForThread.Add(1)
			wrapped(context)
			waitForLocal.Done()
			waitForThread.Wait()
		})

		Context("post-migration event handling", func() {
			var (
				flagSet *flag.FlagSet
			)

			BeforeEach(func() {
				const configFile = "../tests/test_forced_post_migrate_config.yaml"

				flagSet = flag.NewFlagSet("", flag.PanicOnError)
				cli.BoolTFlag{Name: FlagEmitPostMigrationEvent}.Apply(flagSet)
				cli.StringFlag{Name: framework.FlagConfigFile, Value: configFile}.Apply(flagSet)
				cli.DurationFlag{Name: FlagPostMigrationEventTimeout, Value: time.Duration(60) * time.Second}.Apply(flagSet)
			})

			prepareContext := func() *cli.Context {
				context := cli.NewContext(nil, flagSet, &cli.Context{})

				configFile := context.String(framework.FlagConfigFile)
				loadConfig(configFile)
				return context
			}

			expectPostMigrationSucceded := func() {
				sync, err := sync_util.CreateFromConfig(util.GetConfig())
				Expect(err).To(Succeed())

				node, err := sync.Fetch(ctx, "post-migration")
				Expect(err).To(Succeed())
				Expect(node.Value).To(Equal("success"))
				Expect(sync.Delete(ctx, "post-migration", true)).To(Succeed())
			}

			It("Should handle sending post-migration event to forced schemas", func() {
				cli.StringFlag{Name: FlagForcedSchemas, Value: "force_schema"}.Apply(flagSet)
				context := prepareContext()

				actionMigrateWithPostMigrationEvent("up")(context)

				expectPostMigrationSucceded()
			})

			runSchemaMarkingMigration := func(schemaID string) {
				config := util.GetConfig()
				dbType := config.GetString("database/type", "sqlite3")
				dbConnection := config.GetString("database/connection", "")

				db, err := sql.Open(dbType, dbConnection)
				Expect(err).NotTo(HaveOccurred())

				tx, err := db.Begin()
				Expect(err).NotTo(HaveOccurred())

				Expect(migration.MarkSchemaAsModified(schemaID, tx)).To(Succeed())

				Expect(tx.Commit()).To(Succeed())
			}

			runPostMigrationForSchemaID := func(schemaID string) {
				context := prepareContext()

				withinLockedMigration(func(context_pkg.Context, *cli.Context) {
					Expect(migration.Run("up", context.Args())).To(Succeed())
					runSchemaMarkingMigration(schemaID)
					emitPostMigrateEvent(ctx, context.String(FlagForcedSchemas), context.Bool(FlagSyncETCDEvent),
						context.Duration(FlagPostMigrationEventTimeout))
				})(context)
			}

			It("Should handle sending post-migration event to marked schemas", func() {
				runPostMigrationForSchemaID("force_schema")

				Expect(migration.GetModifiedSchemas()).To(BeEmpty())
				expectPostMigrationSucceded()
			})

			It("Should ignore post-migration event for no longer existing schemas", func() {
				runPostMigrationForSchemaID("no_such_schema")

				Expect(migration.GetModifiedSchemas()).To(BeEmpty())
			})
		})

		It("Should not change working directory", func() {
			absoluteConfigPath := path.Join(workingDir(), configPath)
			wrappedFuncCalled := false
			expectedWorkingDir := "/"

			// simply asserting that working dir is the same before and after, would not catch the bug when
			// working dir is already the same as the one used by tested function
			// (e.g. when tested function was executed in a previous test case)
			Expect(workingDir()).NotTo(Equal(expectedWorkingDir))
			Expect(os.Chdir(expectedWorkingDir)).To(Succeed())

			context := getContextWithConfig(absoluteConfigPath, false)
			withinLockedMigration(func(context_pkg.Context, *cli.Context) {
				wrappedFuncCalled = true
			})(context)
			Expect(wrappedFuncCalled).To(BeTrue())

			Expect(workingDir()).To(Equal(expectedWorkingDir))
		})
	})
})

func workingDir() string {
	dir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return dir
}
