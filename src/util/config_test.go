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

package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("Config utility", func() {
	const (
		gohanTestValue = "GOHAN_TEST_VALUE"
	)
	var (
		config *Config
	)

	BeforeEach(func() {
		config = GetConfig()
		os.Setenv("GOHAN_TEST_KEY", "GOHAN_TEST_VALUE")
	})

	Context("When it reads plain YAML config", func() {
		It("Should load proper values", func() {
			Expect(config.ReadConfig("./config_test.yaml")).To(Succeed())
			Expect(config.GetString("unknown_param", "fail")).To(Equal("fail"))
			Expect(config.GetString("keystone/usr_keystone", "fail")).To(Equal("fail"))
			Expect(config.GetString("address", "fail")).To(Equal(":19090"))
			Expect(config.GetBool("keystone/use_keystone", false)).ToNot(BeFalse())
			Expect(config.GetString("database/type", "fail")).To(Equal("sqlite3"))
			etcdServers := config.GetStringList("etcd", nil)
			Expect(etcdServers).ToNot(BeNil())
			etcdServerList := config.GetList("etcd", nil)
			Expect(etcdServerList).ToNot(BeNil())
		})
	})

	Context("When it reads YAML template", func() {
		It("Should load proper values", func() {
			Expect(config.ReadConfig("./config_test.tmpl.yaml")).To(Succeed())
			Expect(config.GetString("unknown_param", "fail")).To(Equal("fail"))
			Expect(config.GetString("keystone/usr_keystone", "fail")).To(Equal("fail"))
			Expect(config.GetString("address", "fail")).To(Equal(":19090"))
			Expect(config.GetBool("keystone/use_keystone", false)).ToNot(BeFalse())
			Expect(config.GetString("database/type", "fail")).To(Equal("sqlite3"))
			Expect(config.GetString("test", "")).To(Equal(gohanTestValue))
			etcdServers := config.GetStringList("etcd", nil)
			Expect(etcdServers).ToNot(BeNil())
			etcdServerList := config.GetList("etcd", nil)
			Expect(etcdServerList).ToNot(BeNil())
		})
	})
})
