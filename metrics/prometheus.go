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
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwan/gohan/util"
	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/rcrowley/go-metrics"
)

const prometheusPath = "metrics/prometheus/"

func getString(config *util.Config, key, defaultValue string) string {
	return config.GetString(prometheusPath+key, defaultValue)
}

func getDuration(config *util.Config, key string, defaultValue time.Duration) time.Duration {
	return config.GetDuration(prometheusPath+key, defaultValue)
}

type prometheusConfig struct {
	flushInterval time.Duration
	timerBuckets  []float64
	namespace     string
	subsystem     string
	serverAddress string
	serverPath    string
	serverBackoff time.Duration
}

type prometheusExporter struct {
	provider *prometheusmetrics.PrometheusConfig
	config   prometheusConfig
}

func (pe *prometheusExporter) Setup(config *util.Config, mainAddress string) error {
	if err := pe.setServerAddress(config, mainAddress); err != nil {
		return err
	}

	pe.config.flushInterval = getDuration(config, "flush_interval", 10*time.Second)
	pe.config.namespace = getString(config, "namespace", "gohan")
	pe.config.subsystem = getString(config, "subsystem", "")

	var err error
	pe.config.timerBuckets, err = getPercentilesFrom(config, prometheusPath+"timer_buckets")
	if err != nil {
		return err
	}

	pe.provider = prometheusmetrics.NewPrometheusProvider(metrics.DefaultRegistry, pe.config.namespace, pe.config.subsystem,
		prometheus.DefaultRegisterer, pe.config.flushInterval)
	pe.provider.WithTimerBuckets(pe.config.timerBuckets)

	pe.config.serverPath = getString(config, "path", "/metrics")
	pe.config.serverBackoff = getDuration(config, "backoff", 5*time.Second)

	return nil
}

func (pe *prometheusExporter) setServerAddress(config *util.Config, mainAddress string) error {
	mainHost, mainPortStr, err := net.SplitHostPort(mainAddress)
	if err != nil {
		return fmt.Errorf("incorrect main gohan address '%s': %s", mainAddress, err)
	}

	mainPort, err := strconv.Atoi(mainPortStr)
	if err != nil {
		return fmt.Errorf("incorrect main gohan address '%s', port is not a number", mainAddress)
	}

	defaultAddress := fmt.Sprintf("%s:%d", mainHost, mainPort+util.DefaultMetricsPortOffset)
	pe.config.serverAddress = getString(config, "address", defaultAddress)

	if pe.config.serverAddress == mainAddress {
		return fmt.Errorf("%s%s cannot be set to the same value main gohan address is: '%s'", prometheusPath, "address",
			mainAddress)
	}

	return nil
}

func (pe *prometheusExporter) IsReady() bool {
	return pe.provider != nil
}

func (pe *prometheusExporter) Start() {
	log.Debug("Starting exposing runtime metrics to prometheus, config %+v", pe.provider)

	pe.serveMetrics()

	go pe.provider.UpdatePrometheusMetrics()
}

func (pe *prometheusExporter) serveMetrics() {
	mux := http.NewServeMux()
	mux.Handle(pe.config.serverPath, promhttp.Handler())

	go func() {
		for {
			err := http.ListenAndServe(pe.config.serverAddress, mux)
			if err != nil {
				log.Critical("Prometheus metrics server error %+v. Trying again.", err)
			}
			time.Sleep(pe.config.serverBackoff)
		}
	}()
}

func (pe *prometheusExporter) GetFlushInterval() time.Duration {
	return pe.provider.FlushInterval
}
