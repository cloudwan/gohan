package healthcheck

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/cloudwan/gohan/util"
)

const (
	healthCheckKey        = "healthcheck"
	healthCheckEnabledKey = healthCheckKey + "/enabled"
	healthCheckEtcdKey    = healthCheckKey + "/etcd_key"
	healthCheckAddressKey = healthCheckKey + "/address"
	healthCheckTimeoutKey = healthCheckKey + "/timeout"
	defaultEtcdKey        = "/gohan"
	defaultTimeout        = 5 * time.Second
)

type HealthCheck struct {
	DataStore db.DB
	Etcd      sync.Sync
	etcdKey   string
	address   string
	timeout   time.Duration
	logger    log.Logger
}

func NewHealthCheck(db db.DB, etcd sync.Sync, serverAddress string, config *util.Config) *HealthCheck {
	if !config.GetBool(healthCheckEnabledKey, false) {
		return nil
	}
	healthCheckAddress, err := getHealthCheckAddress(serverAddress, config)
	if err != nil {
		panic(err)
	}
	timeout := config.GetDuration(healthCheckTimeoutKey, defaultTimeout)
	return &HealthCheck{
		DataStore: db,
		Etcd:      etcd,
		address:   healthCheckAddress,
		timeout:   timeout,
		etcdKey:   config.GetString(healthCheckEtcdKey, defaultEtcdKey),
		logger:    log.NewLogger(log.ModuleName(healthCheckKey)),
	}
}

func (healthCheck *HealthCheck) Run() {
	if healthCheck == nil {
		return
	}
	healthCheckHandler := func(w http.ResponseWriter, req *http.Request) {
		if err := healthCheck.isHealthy(); err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			healthCheck.logger.Error("Health Check error: %s", err.Error())
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", healthCheckHandler)

	go func() {
		for {
			err := http.ListenAndServe(healthCheck.address, mux)
			if err != nil {
				healthCheck.logger.Critical("Health Check server error %+v. Restarting", err)
			}
			time.Sleep(healthCheck.timeout)
		}
	}()
}

func getHealthCheckAddress(serverAddress string, config *util.Config) (string, error) {
	addressAndPort := strings.Split(serverAddress, ":")
	if len(addressAndPort) != 2 {
		return "", errors.New("Incorrect gohan server address: " + serverAddress)
	}
	gohanPort, err := strconv.Atoi(addressAndPort[1])
	if err != nil {
		return "", errors.New("Incorrect gohan server address: expected port number got " + addressAndPort[1])
	}
	healthCheckAddress := config.GetString(healthCheckAddressKey, fmt.Sprintf("%s:%d", addressAndPort[0], gohanPort+1))
	if healthCheckAddress == serverAddress {
		return "", errors.New("HealthCheck address must be different than server address " + serverAddress)
	}
	return healthCheckAddress, nil
}

func (healthCheck *HealthCheck) isHealthy() error {
	if dbErr := db.WithinTx(healthCheck.DataStore, func(transaction transaction.Transaction) error { return nil }); dbErr != nil {
		return dbErr
	}
	if _, syncErr := healthCheck.Etcd.Fetch(healthCheck.etcdKey); syncErr != nil && syncErr != etcdv3.KeyNotFound {
		return syncErr
	}
	return nil
}
