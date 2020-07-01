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
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics utils", func() {
	var (
		emptyConfig *util.Config
	)

	BeforeEach(func() {
		emptyConfig = util.NewConfig(map[string]interface{}{})
	})

	Context("Percentiles", func() {
		const key = "some_key"

		thenPercentilesAre := func(config *util.Config, expectedPercentiles []float64) {
			percentiles, err := getPercentilesFrom(config, key)

			Expect(err).NotTo(HaveOccurred())
			Expect(percentiles).To(Equal(expectedPercentiles))
		}

		It("Returns default percentile value by default", func() {
			thenPercentilesAre(emptyConfig, []float64{0.5, 0.75, 0.95, 0.99, 0.999})
		})

		givenConfigWithPercentiles := func(percentiles []interface{}) *util.Config {
			return util.NewConfig(map[string]interface{}{
				key: percentiles,
			})
		}

		It("Returns percentiles from config if present", func() {
			config := givenConfigWithPercentiles([]interface{}{"0.32", "0.64"})

			thenPercentilesAre(config, []float64{0.32, 0.64})
		})

		It("Fails if can't convert to a number", func() {
			config := givenConfigWithPercentiles([]interface{}{"0.5", "not_a_number"})

			_, err := getPercentilesFrom(config, key)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("Exporter factory", func() {
		// NOTE: dynamic type checking in tests is far from ideal...
		thenCreatedExporterIsOfType := func(config *util.Config, expectedType interface{}) {
			exporter, err := createMetricsExporter(config)

			Expect(err).NotTo(HaveOccurred())
			Expect(exporter).To(BeAssignableToTypeOf(expectedType))
		}

		It("Creates a Graphite exporter by default", func() {
			thenCreatedExporterIsOfType(emptyConfig, &graphiteExporter{})
		})

		givenConfigWithExporter := func(exporterType string) *util.Config {
			return util.NewConfig(map[string]interface{}{
				"metrics": map[string]interface{}{
					"exporter": exporterType,
				},
			})
		}

		DescribeTable("Create configured exporter",
			func(exporterTag string, expectedType interface{}) {
				config := givenConfigWithExporter(exporterTag)
				thenCreatedExporterIsOfType(config, expectedType)
			},
			Entry("Graphite", graphiteTag, &graphiteExporter{}),
			Entry("Prometheus", prometheusTag, &prometheusExporter{}),
		)

		It("Fails if unknown exporter", func() {
			config := givenConfigWithExporter("unknown")

			_, err := createMetricsExporter(config)

			Expect(err).To(HaveOccurred())
		})
	})
})
