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

package goplugin_test

import (
	"io"
	"os"
	"time"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/otto"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JS integration", func() {
	var (
		env              *goplugin.Environment
		schemaManager    *schema.Manager
		extensionManager *extension.Manager
		logWriter        io.Writer = os.Stderr
	)

	const (
		schemaPath = "test_data/test_schema.yaml"
	)

	BeforeSuite(func() {
		l.SetUpBasicLogging(logWriter, l.DefaultFormat)
	})

	initGlobalObjects := func() {
		schemaManager = schema.GetManager()
		Expect(schemaManager.LoadSchemaFromFile(schemaPath)).To(Succeed())

		extensionManager = extension.GetManager()
	}

	initGoExtension := func() {
		env = goplugin.NewEnvironment("test", nil, nil)

		err := env.Load("test_data/ext_good/ext_good.so")
		Expect(err).To(BeNil())
		Expect(env.Start()).To(Succeed())
	}

	initOttoExtension := func() {
		ottoEnv := otto.NewEnvironment("otto", nil, nil, nil)
		ottoCode, err := schema.NewExtension(map[string]interface{}{
			"id": "test_otto_extension",
			"code": `gohan_register_handler("js_throw_custom_exception", function(context) {
			      	 	gohan_log_debug("JS throwing handler invoked");
						context.js_invoked = true;
      					throw new CustomException("test exception", 456);
					});
			`,
			"path": ".*",
		})
		Expect(err).ToNot(HaveOccurred())
		ottoCode.URL = "good_extension.js"

		extensions := []*schema.Extension{ottoCode}
		timeLimit := time.Duration(1) * time.Minute
		timeLimits := []*schema.PathEventTimeLimit{}
		Expect(ottoEnv.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test")).To(Succeed())

		extensionManager.RegisterEnvironment("test", ottoEnv)
	}

	BeforeEach(func() {
		initGlobalObjects()
		initGoExtension()
		initOttoExtension()

		testSchema := env.Schemas().Find("test")
		Expect(testSchema).NotTo(BeNil())
	})

	AfterEach(func() {
		env.Stop()
		schemaManager.ClearExtensions()
		extensionManager.UnRegisterEnvironment("test")
	})

	Context("Error propagation", func() {
		var (
			ctx goext.Context
			err error
		)

		BeforeEach(func() {
			ctx = map[string]interface{}{
				"schema_id": "test",
			}

			err = env.Core().TriggerEvent("js_throw_custom_exception", ctx)
		})

		It("should propagate errors", func() {
			Expect(err).To(HaveOccurred())
			Expect(ctx).To(HaveKeyWithValue("js_invoked", true))
		})

		It("should clean context", func() {
			Expect(ctx).NotTo(HaveKey("exception"))
			Expect(ctx).NotTo(HaveKey("exception_message"))
		})

		It("should propagate error code", func() {
			Expect(err.(*goext.Error).Status).To(Equal(456))
		})

		It("should propagate error message", func() {
			Expect(err.(*goext.Error).Error()).To(ContainSubstring("test exception"))
		})
	})
})
