package metrics

import (
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"

	"fmt"
	"net"
	"strconv"
	"time"
)

var (
	graphiteConfigs   []graphite.Config
	monitoringEnabled bool
	log               = l.NewLogger()
)

func getGraphitePercentiles(config *util.Config) (percentiles []float64, err error) {
	defaultPercentiles := []string{"0.5", "0.75", "0.95", "0.99", "0.999"}
	percentilesStr := config.GetStringList("metrics/graphite/percentiles", defaultPercentiles)
	percentiles = make([]float64, len(percentilesStr))
	for i, v := range percentilesStr {
		if percentiles[i], err = strconv.ParseFloat(v, 64); err != nil {
			return nil, fmt.Errorf("Error '%s' when parsing metrics/graphite/percentiles, expecting a float, '%s' given", err, v)
		}
	}

	return percentiles, nil
}

func getGraphiteConfig(config *util.Config) (graphiteConfigs []graphite.Config, err error) {
	graphiteEndpoints := config.GetStringList("metrics/graphite/endpoints", []string{})
	if len(graphiteEndpoints) == 0 {
		log.Debug("No graphite endpoints set in config file")
		return graphiteConfigs, nil
	}

	var baseconfig graphite.Config

	if baseconfig.Percentiles, err = getGraphitePercentiles(config); err != nil {
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

func SetupMetrics(config *util.Config) (err error) {
	monitoringEnabled = config.GetBool("metrics/enabled", false)
	graphiteConfigs, err = getGraphiteConfig(config)
	return
}

func StartMetricsProcess() {
	if monitoringEnabled && len(graphiteConfigs) > 0 {
		log.Debug("Starting sending runtime metrics to graphite, config %+v", graphiteConfigs)
		metrics.RegisterRuntimeMemStats(graphiteConfigs[0].Registry)
		go metrics.CaptureRuntimeMemStats(graphiteConfigs[0].Registry, graphiteConfigs[0].FlushInterval)
		for _, config := range graphiteConfigs {
			go graphite.WithConfig(config)
		}

	}
}

func UpdateTimer(since time.Time, format string, args ...interface{}) {
	if monitoringEnabled {
		m := metrics.GetOrRegisterTimer(fmt.Sprintf(format, args...), metrics.DefaultRegistry)
		m.UpdateSince(since)
	}
}
