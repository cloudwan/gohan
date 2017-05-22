package transaction

import (
	"time"

	"strings"

	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/jmoiron/sqlx"
)

const NUMBER_OF_RETRIES_CONFIG_KEY = "database/transaction_retries/attempts"
const RETRY_STRATEGY_CONFIG_KEY = "database/transaction_retries/strategy"
const INTERVAL_BETWEEN_ATTEMPTS_CONFIG_KEY = "database/transaction_retries/interval_between_attempts"
const DEFAULT_NUMBER_OF_RETRIES = 1
const DEFAULT_RETRY_STRATEGY = "deadlock"
const DEFAULT_INTERVAL_BETWEEN_ATTEMPTS = "100ms"

const MYSQL_DEADLOCK_MSG = "Error 1213: Deadlock found when trying to get lock; try restarting transaction"
const SQLITE_DEADLOCK_MSG = "database is locked"

type RetryConfig struct {
	Attempts      int
	Strategy      RetryStrategyPredicate
	SleepInterval time.Duration
}

var cache RetryConfig
var cacheInitialized = false

type RetryableTransaction struct {
	Tx     Transaction
	Config RetryConfig
}

type RetryStrategyPredicate func(error) bool

var strategies = map[string]RetryStrategyPredicate{
	"deadlock": IsDeadlock,
}

func NewRetryableTransaction(transaction Transaction) Transaction {
	config := readConfig()
	tx := &RetryableTransaction{
		Tx:     transaction,
		Config: config,
	}
	log.Debug("Created retryable transaction tx: %v, rawTx: %v, attempts: %d, interval: %s", &tx, transaction.RawTransaction(), config.Attempts, config.SleepInterval)
	return tx
}

func IsDeadlock(err error) bool {
	return strings.Contains(err.Error(), MYSQL_DEADLOCK_MSG) || strings.Contains(err.Error(), SQLITE_DEADLOCK_MSG)
}

func readConfig() RetryConfig {
	if cacheInitialized {
		return cache
	}
	config := util.GetConfig()
	attempts := config.GetInt(NUMBER_OF_RETRIES_CONFIG_KEY, DEFAULT_NUMBER_OF_RETRIES)
	configStrategy := config.GetString(RETRY_STRATEGY_CONFIG_KEY, DEFAULT_RETRY_STRATEGY)
	retryStrategy, ok := strategies[configStrategy]
	if !ok {
		log.Error("%s Strategy isn't implemented, using default one: %s", configStrategy, DEFAULT_INTERVAL_BETWEEN_ATTEMPTS)
		retryStrategy = strategies[DEFAULT_INTERVAL_BETWEEN_ATTEMPTS]
	}
	sleepInterval, parseErr := time.ParseDuration(config.GetString(INTERVAL_BETWEEN_ATTEMPTS_CONFIG_KEY, DEFAULT_INTERVAL_BETWEEN_ATTEMPTS))
	if parseErr != nil {
		log.Error("Failed to parse retry interval from config, using default one: %s", DEFAULT_INTERVAL_BETWEEN_ATTEMPTS)
		sleepInterval, _ = time.ParseDuration(DEFAULT_INTERVAL_BETWEEN_ATTEMPTS)
	}
	cache = RetryConfig{
		Attempts:      attempts,
		Strategy:      retryStrategy,
		SleepInterval: sleepInterval,
	}
	cacheInitialized = true
	return cache
}

func (t *RetryableTransaction) retryOnError(f func() error) (err error) {
	for attempt := 1; true; attempt++ {
		err = f()
		if err != nil && t.Config.Strategy(err) && (t.Config.Attempts < 0 || attempt < t.Config.Attempts) {
			log.Debug("Retrying tx: %v, attempt %d/%d, sleeping %s", &t, attempt+1, t.Config.Attempts, t.Config.SleepInterval)
			time.Sleep(t.Config.SleepInterval)
		} else {
			return
		}
	}
	return
}

func (t *RetryableTransaction) Create(resource *schema.Resource) error {
	return t.retryOnError(func() error { return t.Tx.Create(resource) })
}

func (t *RetryableTransaction) Update(resource *schema.Resource) error {
	return t.retryOnError(func() error { return t.Tx.Update(resource) })
}
func (t *RetryableTransaction) StateUpdate(resource *schema.Resource, state *ResourceState) error {
	return t.retryOnError(func() error { return t.Tx.StateUpdate(resource, state) })
}
func (t *RetryableTransaction) Delete(schema *schema.Schema, resourceID interface{}) error {
	return t.retryOnError(func() error { return t.Tx.Delete(schema, resourceID) })
}
func (t *RetryableTransaction) Fetch(schema *schema.Schema, filter Filter) (resource *schema.Resource, err error) {
	t.retryOnError(func() error {
		resource, err = t.Tx.Fetch(schema, filter)
		return err
	})
	return
}
func (t *RetryableTransaction) LockFetch(schema *schema.Schema, filter Filter, lockPolicy schema.LockPolicy) (resource *schema.Resource, err error) {
	t.retryOnError(func() error {
		resource, err = t.Tx.LockFetch(schema, filter, lockPolicy)
		return err
	})
	return
}
func (t *RetryableTransaction) StateFetch(schema *schema.Schema, filter Filter) (state ResourceState, err error) {
	t.retryOnError(func() error {
		state, err = t.Tx.StateFetch(schema, filter)
		return err
	})
	return

}
func (t *RetryableTransaction) List(schema *schema.Schema, filter Filter, listOptions *ListOptions, pg *pagination.Paginator) (resources []*schema.Resource, total uint64, err error) {
	t.retryOnError(func() error {
		resources, total, err = t.Tx.List(schema, filter, listOptions, pg)
		return err
	})
	return
}
func (t *RetryableTransaction) LockList(schema *schema.Schema, filter Filter, listOptions *ListOptions, pg *pagination.Paginator, lockPolicy schema.LockPolicy) (resources []*schema.Resource, total uint64, err error) {
	t.retryOnError(func() error {
		resources, total, err = t.Tx.LockList(schema, filter, listOptions, pg, lockPolicy)
		return err
	})
	return
}
func (t *RetryableTransaction) RawTransaction() *sqlx.Tx {
	return t.Tx.RawTransaction()

}
func (t *RetryableTransaction) Query(schema *schema.Schema, query string, arguments []interface{}) (list []*schema.Resource, err error) {
	t.retryOnError(func() error {
		list, err = t.Tx.Query(schema, query, arguments)
		return err
	})
	return
}
func (t *RetryableTransaction) Commit() error {
	return t.Tx.Commit()
}
func (t *RetryableTransaction) Exec(query string, args ...interface{}) error {
	return t.retryOnError(func() error { return t.Tx.Exec(query, args) })
}
func (t *RetryableTransaction) Close() error {
	return t.Tx.Close()
}
func (t *RetryableTransaction) Closed() bool {
	return t.Tx.Closed()
}
func (t *RetryableTransaction) GetIsolationLevel() Type {
	return t.Tx.GetIsolationLevel()
}
