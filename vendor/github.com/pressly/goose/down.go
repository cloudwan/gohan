package goose

import (
	"database/sql"
	"fmt"
)

func Down(db *sql.DB, dir string) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}

	return current.Down(db)
}

func DownTo(db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	for {
		currentVersion, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		prev, err := migrations.Previous(currentVersion)
		if err != nil {
			if err == ErrNoNextVersion {
				fmt.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
				return nil
			}
			return err
		}

		if prev.Version < version {
			fmt.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		current, err := migrations.Current(currentVersion)
		if err != nil {
			return fmt.Errorf("no migration %v", currentVersion)
		}

		if err = current.Down(db); err != nil {
			return err
		}
	}

	return nil
}
