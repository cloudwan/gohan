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
	"time"

	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
	metrics "github.com/rcrowley/go-metrics"
)

var (
	exporter          metricsExporter
	monitoringEnabled bool
	log               = l.NewLogger()
)

type metricsExporter interface {
	Setup(config *util.Config, serverAddress string) error
	IsReady() bool
	Start()
	GetFlushInterval() time.Duration
}

// SetupMetrics setups metrics from config
func SetupMetrics(config *util.Config, serverAddress string) (err error) {
	monitoringEnabled = config.GetBool("metrics/enabled", false)

	exporter, err = createMetricsExporter(config)
	if err != nil {
		return err
	}

	return exporter.Setup(config, serverAddress)
}

// StartMetricsProcess starts to send runtime metrics to chosen exporter
func StartMetricsProcess() {
	if !monitoringEnabled || !exporter.IsReady() {
		return
	}
	metrics.RegisterRuntimeMemStats(metrics.DefaultRegistry)
	go metrics.CaptureRuntimeMemStats(metrics.DefaultRegistry, exporter.GetFlushInterval())

	exporter.Start()
}

// UpdateTimer updates metrics timer
func UpdateTimer(since time.Time, format string, args ...interface{}) {
	if monitoringEnabled {
		m := metrics.GetOrRegisterTimer(fmt.Sprintf(format, args...), metrics.DefaultRegistry)
		m.UpdateSince(since)
	}
}

func UpdateCounter(delta int64, format string, args ...interface{}) {
	if monitoringEnabled {
		m := metrics.GetOrRegisterCounter(fmt.Sprintf(format, args...), metrics.DefaultRegistry)
		m.Inc(delta)
	}
}

func UpdateGauge(value int64, format string, args ...interface{}) {
	if monitoringEnabled {
		m := metrics.GetOrRegisterGauge(fmt.Sprintf(format, args...), metrics.DefaultRegistry)
		m.Update(value)
	}
}
