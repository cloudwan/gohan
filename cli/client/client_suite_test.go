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

package client_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cli Suite")
}

var _ = Describe("Suit set up and tear down", func() {
	var _ = BeforeSuite(func() {
		os.Unsetenv("OS_AUTH_URL")
		os.Unsetenv("OS_USERNAME")
		os.Unsetenv("OS_USERID")
		os.Unsetenv("OS_PASSWORD")
		os.Unsetenv("OS_TENANT_ID")
		os.Unsetenv("OS_TENANT_NAME")
		os.Unsetenv("OS_DOMAIN_ID")
		os.Unsetenv("OS_DOMAIN_NAME")
	})
})
