package backup

import (
	"io"
	"fmt"
)

// Backup backups database to w.
func Backup(dbType, dsn string, w io.Writer) error {
	switch dbType {
	case "mysql":
		return doBackupMySQL(dsn, w)
	default:
		return fmt.Errorf("unsupported database type %q", dbType)
	}
}

func doBackupMySQL(dsn string, w io.Writer) error {
	config, err := NewMySQLConfig(dsn)
	if err != nil {
		return fmt.Errorf("Invalid connection string %q: %s", dsn, err)
	}

	return MySQL(config, w)
}
