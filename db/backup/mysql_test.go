package backup

import (
	"reflect"
	"testing"
)

func TestNewMySQLConfig(t *testing.T) {
	data := []struct {
		dsn      string
		expected *MySQLConfig
	}{
		{
			dsn: "user:password@/database",
			expected: &MySQLConfig{
				Host:     "127.0.0.1",
				Port:     "3306",
				DB:       "database",
				User:     "user",
				Password: "password",
			},
		},
		{
			dsn: "user:password@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true",
			expected: &MySQLConfig{
				Host:     "localhost",
				Port:     "5555",
				DB:       "dbname",
				User:     "user",
				Password: "password",
			},
		},
		{
			dsn: "user:password@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci",
			expected: &MySQLConfig{
				Host:     "de:ad:be:ef::ca:fe",
				Port:     "80",
				DB:       "dbname",
				User:     "user",
				Password: "password",
			},
		},
		{
			dsn: "id:password@broken/dbname",
		},
		{
			dsn: "broken/dbname",
		},
	}

	for _, tt := range data {
		config, err := NewMySQLConfig(tt.dsn)
		if err != nil {
			t.Log("Unexpected error", tt, err)
		}
		if !reflect.DeepEqual(config, tt.expected) {
			t.Error(tt, "got", config)
		}
	}
}
