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
	"net"
	"time"

	"github.com/cloudwan/gohan/util"
	graphite "github.com/cyberdelia/go-metrics-graphite"
	metrics "github.com/rcrowley/go-metrics"
)

type graphiteExporter struct {
	configs []graphite.Config
}

func (ge *graphiteExporter) Setup(config *util.Config, _ string) error {
	configs, err := ge.getGraphiteConfig(config)
	if err != nil {
		return err
	}

	ge.configs = configs
	return nil
}

func (ge *graphiteExporter) IsReady() bool {
	return len(ge.configs) > 0
}

func (ge *graphiteExporter) Start() {
	log.Debug("Starting sending runtime metrics to graphite, config %+v", ge.configs)

	for _, config := range ge.configs {
		go graphite.WithConfig(config)
	}
}

func (ge *graphiteExporter) getGraphiteConfig(config *util.Config) (graphiteConfigs []graphite.Config, err error) {
	graphiteEndpoints := config.GetStringList("metrics/graphite/endpoints", []string{})
	if len(graphiteEndpoints) == 0 {
		log.Debug("No graphite endpoints set in config file")
		return graphiteConfigs, nil
	}

	var baseconfig graphite.Config

	if baseconfig.Percentiles, err = getPercentilesFrom(config, "metrics/graphite/percentiles"); err != nil {
		return nil, err
	}
	baseconfig.FlushInterval = time.Duration(config.GetInt("metrics/graphite/flush_interval_sec", 60)) * time.Second
	baseconfig.Prefix = config.GetString("metrics/graphite/prefix", "gohan")
	baseconfig.DurationUnit = time.Nanosecond
	baseconfig.Registry = metrics.DefaultRegistry

	for _, endpoint := range graphiteEndpoints {
		addr, err := net.ResolveTCPAddr("tcp", endpoint)
		if err != nil {
			return nil, fmt.Errorf("Can't resolve graphite endpoint %s: %s", endpoint, err)
		}
		config := baseconfig
		config.Addr = addr
		graphiteConfigs = append(graphiteConfigs, config)
	}

	return graphiteConfigs, nil
}

func (ge *graphiteExporter) GetFlushInterval() time.Duration {
	return ge.configs[0].FlushInterval
}
