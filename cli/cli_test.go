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
	"flag"
	"fmt"
	"github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/codegangsta/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"sync"
	"time"
)

const useEtcdEnv = "USE_ETCD_DURING_MIGRATIONS"

func getContextWithConfig(configFile string, useEtcd bool) *cli.Context {
	if useEtcd {
		os.Setenv(useEtcdEnv, "true")
	}

	configFlag := cli.StringFlag{Name: "config-file", Value: configFile}
	useEtcdFlag := cli.BoolFlag{Name: lockWithEtcd, EnvVar: useEtcdEnv}

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
		waitForThread sync.WaitGroup
		waitForLocal  sync.WaitGroup
		etcdSync      *etcdv3.Sync
	)

	BeforeEach(func() {
		waitForThread = sync.WaitGroup{}
		waitForLocal = sync.WaitGroup{}

		waitForThread.Add(1)
		waitForLocal.Add(1)

		var err error
		etcdSync, err = etcdv3.NewSync([]string{etcdServer}, time.Second)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.Unsetenv(useEtcdEnv)
	})

	Describe("Post migration subcommand wrapper tests", func() {
		It("Should lock when the flag is set - migrationsSubCommand wrapper", func() {
			lock := func(context *cli.Context) {
				waitForThread.Done()
				waitForLocal.Wait()
			}

			context := getContextWithConfig(configPath, true)
			Expect(context.String("config-file")).To(Equal(configPath))

			wrapped := func() {
				migrationSubcommandWithLock(lock)(context)
				waitForThread.Done()
			}

			go wrapped()
			waitForThread.Wait()
			waitForThread.Add(1)
			_, err := etcdSync.Lock(syncMigrationsPath, false)
			waitForLocal.Done()
			waitForThread.Wait()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("failed to lock path %s", syncMigrationsPath)))
		})

		It("Should not lock when the flag is unset - migrationsSubCommand wrapper", func() {
			lock := func() {
				etcdSync.Lock(syncMigrationsPath, true)
				waitForThread.Done()
				waitForLocal.Wait()
				etcdSync.Unlock(syncMigrationsPath)
				waitForThread.Done()
			}

			wrapped := migrationSubcommandWithLock(func(context *cli.Context) {})
			context := getContextWithConfig(configPath, false)
			Expect(context.String("config-file")).To(Equal(configPath))

			go lock()
			waitForThread.Wait()
			waitForThread.Add(1)
			wrapped(context)
			waitForLocal.Done()
			waitForThread.Wait()
		})
	})
})
