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

const etcdTimeoutDefaultValue = 1000

// CreateFromConfig creates etcd sync from config
func CreateFromConfig(config *util.Config) (s sync.Sync, err error) {
	syncType := config.GetString("sync", "etcd")
	switch syncType {
	case "etcd":
		etcdServers := config.GetStringList("etcd", nil)
		if etcdServers != nil {
			log.Info("etcd servers: %s", etcdServers)
			s = etcd.NewSync(etcdServers)
		}
	case "etcdv3":
		etcdServers := config.GetStringList("etcd", nil)
		if etcdServers != nil {
			log.Info("etcd servers: %s", etcdServers)
			s, err = etcdv3.NewSync(etcdServers, time.Duration(config.GetInt("etcd_timeout", etcdTimeoutDefaultValue))*time.Millisecond)
			if err != nil {
				err = fmt.Errorf("failed to connect to etcd servers: %s", err)
				return
			}
		}
	default:
		err = fmt.Errorf("invalid sync type: %s", syncType)
		return
	}
	return
}
