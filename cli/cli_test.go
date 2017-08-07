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
	"sync"
	"time"
)

func getContextWithConfig(configFile string) *cli.Context {
	configFlag := cli.StringFlag{Name: "config-file", Value: configFile}
	set := flag.NewFlagSet("", flag.ContinueOnError)
	configFlag.Apply(set)
	return cli.NewContext(nil, set, &cli.Context{})
}

var _ = Describe("CLI", func() {
	Describe("Post migration subcommand wrapper tests", func() {
		It("Should lock - migrationsSubCommand wrapper", func() {
			const (
				configPath = "../tests/test_etcd.yaml"
				etcdServer = "http://127.0.0.1:2379"
			)

			waitForLock := sync.WaitGroup{}
			waitForFail := sync.WaitGroup{}

			lock := func(context *cli.Context) {
				waitForLock.Done()
				waitForFail.Wait()
			}

			etcdSync, err := etcdv3.NewSync([]string{etcdServer}, time.Second)
			Expect(err).ToNot(HaveOccurred())

			wrapped := migrationSubcommandWithLock(lock)
			context := getContextWithConfig(configPath)
			Expect(context.String("config-file")).To(Equal(configPath))

			waitForFail.Add(1)
			waitForLock.Add(1)
			go wrapped(context)
			waitForLock.Wait()
			_, err = etcdSync.Lock(syncMigrationsPath, false)
			waitForFail.Done()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("failed to lock path %s", syncMigrationsPath)))
		})
	})
})
