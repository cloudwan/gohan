package healthcheck

import (
	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/sync"
)

type Healthcheck struct {
	DataStore db.DB
	Etcd      sync.Sync
	etcdKey   string
}

func (healthcheck Healthcheck) IsHealthy() error {
	if dbErr := db.WithinTx(healthcheck.DataStore, func(transaction transaction.Transaction) error { return nil }); dbErr != nil {
		return dbErr
	}
	if _, syncErr := healthcheck.Etcd.Fetch(healthcheck.etcdKey); syncErr != nil {
		return syncErr
	}
	return nil
}

func NewHealthcheck(db db.DB, etcd sync.Sync, etcdKey string) *Healthcheck {
	return &Healthcheck{
		DataStore: db,
		Etcd:      etcd,
		etcdKey:   etcdKey,
	}
}
