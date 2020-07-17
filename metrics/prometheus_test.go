// Copyright (C) 2020 NTT Innovation Institute, Inc.
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

package metrics

import (
	"time"

	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prometheus metrics", func() {
	var (
		pe          *prometheusExporter
		emptyConfig *util.Config
	)

	BeforeEach(func() {
		pe = &prometheusExporter{}
		emptyConfig = util.NewConfig(map[string]interface{}{})
	})

	It("Has reasonable defaults after setup", func() {
		Expect(pe.Setup(emptyConfig, ":1234")).To(Succeed())

		Expect(pe.config.flushInterval).To(Equal(10 * time.Second))
		Expect(pe.config.timerBuckets).To(Equal([]float64{0.5, 0.75, 0.95, 0.99, 0.999}))
		Expect(pe.config.namespace).To(Equal("gohan"))
		Expect(pe.config.subsystem).To(Equal(""))
		Expect(pe.config.serverPath).To(Equal("/metrics"))
		Expect(pe.config.serverBackoff).To(Equal(5 * time.Second))
	})

	It("Default server address is based on main address with port+2", func() {
		Expect(pe.Setup(emptyConfig, "192.168.42.88:1234")).To(Succeed())

		Expect(pe.config.serverAddress).To(Equal("192.168.42.88:1236"))
	})

	It("Rejects invalid main address", func() {
		Expect(pe.Setup(emptyConfig, ":not_a_number")).NotTo(Succeed())
		Expect(pe.Setup(emptyConfig, "192.168.0.1:12:34")).NotTo(Succeed())
	})

	It("Rejects server address equal to main adress", func() {
		const address = "192.168.1.2:4321"

		config := util.NewConfig(map[string]interface{}{
			"metrics": map[string]interface{}{
				"prometheus": map[string]interface{}{
					"address": address,
				},
			},
		})

		Expect(pe.Setup(config, address)).NotTo(Succeed())
	})

	It("Reads from config during setup", func() {
		config := util.NewConfig(map[string]interface{}{
			"metrics": map[string]interface{}{
				"prometheus": map[string]interface{}{
					"flush_interval": "123s",
					"timer_buckets":  []interface{}{"0.42", "0.88"},
					"namespace":      "namespace",
					"subsystem":      "subsystem",
					"address":        "192.168.1.2:4321",
					"path":           "/testmetrics",
					"backoff":        "42s",
				},
			},
		})

		Expect(pe.Setup(config, ":1234")).To(Succeed())

		Expect(pe.config.flushInterval).To(Equal(123 * time.Second))
		Expect(pe.config.timerBuckets).To(Equal([]float64{0.42, 0.88}))
		Expect(pe.config.namespace).To(Equal("namespace"))
		Expect(pe.config.subsystem).To(Equal("subsystem"))
		Expect(pe.config.serverAddress).To(Equal("192.168.1.2:4321"))
		Expect(pe.config.serverPath).To(Equal("/testmetrics"))
		Expect(pe.config.serverBackoff).To(Equal(42 * time.Second))
	})

	It("Is not ready without setup", func() {
		Expect(pe.IsReady()).To(BeFalse())
	})

	It("Is ready after successful setup", func() {
		Expect(pe.Setup(emptyConfig, ":1234")).To(Succeed())

		Expect(pe.IsReady()).To(BeTrue())
	})

	It("Is possible to access flush interval after setup", func() {
		config := util.NewConfig(map[string]interface{}{
			"metrics": map[string]interface{}{
				"prometheus": map[string]interface{}{
					"flush_interval": "123s",
				},
			},
		})

		Expect(pe.Setup(config, ":1234")).To(Succeed())

		Expect(pe.GetFlushInterval()).To(Equal(123 * time.Second))
	})
})
