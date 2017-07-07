package util

import (
	"fmt"
	"time"

	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/sync/etcd"
	"github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/cloudwan/gohan/util"
)

var log = l.NewLogger()

// CreateFromConfig creates etcd sync from config
func CreateFromConfig(config *util.Config) (etcdSync sync.Sync, err error) {
	syncType := config.GetString("sync", "etcd")
	etcdServers := config.GetStringList("etcd", nil)
	if etcdServers == nil {
		err = fmt.Errorf("no etcd found in config file")
		return
	}
	switch syncType {
	case "etcd":
		log.Info("etcd servers: %s", etcdServers)
		etcdSync = etcd.NewSync(etcdServers)
	case "etcdv3":
		log.Info("etcd servers: %s", etcdServers)
		etcdSync, err = etcdv3.NewSync(etcdServers, time.Second)
		if err != nil {
			err = fmt.Errorf("failed to connect to etcd servers: %s", err)
			return
		}
	default:
		err = fmt.Errorf("invalid sync type: %s", syncType)
		return
	}
	return
}
