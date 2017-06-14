package db

import (
	"github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
)

func IsDeadlockError(err error) bool {
	switch t := err.(type) {
	case *mysql.MySQLError:
		switch t.Number {
		case 1213:
			return true
		}
	}

	return false
}

func IsDuplicateEntryError(err error) bool {
	switch t := err.(type) {
	case *mysql.MySQLError:
		switch t.Number {
		case 1062:
			return true
		}
	case sqlite3.Error:
		switch t.ExtendedCode {
		case sqlite3.ErrConstraintUnique:
			return true
		}
	}

	return false
}
