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

package golang_test

import (
	"time"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/golang"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gohanscript extension manager", func() {
	var (
		timeLimit  time.Duration
		timeLimits []*schema.PathEventTimeLimit
	)

	BeforeEach(func() {
		timeLimit = time.Duration(10) * time.Second
		timeLimits = []*schema.PathEventTimeLimit{}
	})

	AfterEach(func() {
		extension.ClearManager()
	})

	Describe("Loading an extension", func() {
		Context("When extension uses code property", func() {
			It("should run the extension code", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id":        "test_extension",
					"code":      "test_callback",
					"code_type": "go",
					"path":      ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := golang.NewEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["person"]).ToNot(BeNil())
			})
		})
	})

})
