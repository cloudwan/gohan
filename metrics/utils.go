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
	"fmt"
	"strconv"

	"github.com/cloudwan/gohan/util"
)

func getPercentilesFrom(config *util.Config, path string) (percentiles []float64, err error) {
	defaultPercentiles := []string{"0.5", "0.75", "0.95", "0.99", "0.999"}
	percentilesStr := config.GetStringList(path, defaultPercentiles)
	percentiles = make([]float64, len(percentilesStr))
	for i, v := range percentilesStr {
		if percentiles[i], err = strconv.ParseFloat(v, 64); err != nil {
			return nil, fmt.Errorf("error '%s' when parsing %s, expecting a float, '%s' given", err, path, v)
		}
	}

	return percentiles, nil
}

const (
	graphiteTag   = "graphite"
	prometheusTag = "prometheus"
)

func createMetricsExporter(config *util.Config) (metricsExporter, error) {
	exporterTag := config.GetString("metrics/exporter", graphiteTag)

	switch exporterTag {
	case graphiteTag:
		return &graphiteExporter{}, nil
	case prometheusTag:
		return &prometheusExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported metrics exporter under metrics/exporter: %s, must be 'graphite' or 'prometheus'",
			exporterTag)
	}
}
