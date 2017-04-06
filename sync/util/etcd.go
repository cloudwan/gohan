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
			s, err = etcdv3.NewSync(etcdServers, time.Second)
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
