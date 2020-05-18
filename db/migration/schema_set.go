// Copyright (C) 2020 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migration

import (
	"database/sql"
	"errors"
)

type stringSet map[string]bool

type schemaSetBackend interface {
	loadSchemaIDs() (stringSet, error)
	persistSchemaID(schemaID string, tx *sql.Tx) error
	deleteSchemaID(schemaID string) error
}

var errNilBackend = errors.New("schemaSetBackend is nil")

type schemaSet struct {
	schemas stringSet
	backend schemaSetBackend
}

func newSchemaSet() *schemaSet {
	return &schemaSet{
		schemas: stringSet{},
		backend: nil, // there is a global schemaSet instance, backend must be lazily initialized
	}
}

func (s *schemaSet) init(backend schemaSetBackend) (err error) {
	log.Notice("post-migration: initializing schema ID set")

	if backend == nil {
		return errNilBackend
	}

	s.backend = backend

	schemas, err := backend.loadSchemaIDs()
	if err != nil {
		return err
	}

	s.schemas = schemas

	return nil
}

func (s *schemaSet) markSchemaID(schemaID string, tx *sql.Tx) (err error) {
	log.Info("post-migration: marking schema '%s'", schemaID)

	if s.backend == nil {
		return errNilBackend
	}

	err = s.backend.persistSchemaID(schemaID, tx)
	if err != nil {
		return err
	}

	s.schemas[schemaID] = true
	return nil
}

func (s *schemaSet) getSchemaIDs() []string {
	schemas := make([]string, 0, len(s.schemas))
	for schema := range s.schemas {
		schemas = append(schemas, schema)
	}
	return schemas
}

func (s *schemaSet) removeSchemaID(schemaID string) error {
	log.Info("post-migration: removing schema '%s'", schemaID)

	if s.backend == nil {
		return errNilBackend
	}

	err := s.backend.deleteSchemaID(schemaID)
	if err != nil {
		return err
	}

	delete(s.schemas, schemaID)
	return nil
}

type postMigrationEventsBackend struct {
	db *sql.DB
}

var _ schemaSetBackend = &postMigrationEventsBackend{}

func newSchemaSetDbBackend(db *sql.DB) *postMigrationEventsBackend {
	return &postMigrationEventsBackend{db: db}
}

const schemaSetDbTable = "post_migration_events"

func (b *postMigrationEventsBackend) loadSchemaIDs() (stringSet, error) {
	log.Debug("post-migration: preloading pending schema IDs")

	tx, err := b.db.Begin()
	if err != nil {
		return nil, err
	}

	schemas, err := b.readSchemaIDsInTx(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return schemas, nil
}

func (b *postMigrationEventsBackend) readSchemaIDsInTx(tx *sql.Tx) (stringSet, error) {
	if err := b.ensureTableExists(tx); err != nil {
		return nil, err
	}

	const query = "SELECT `schema_id` FROM `" + schemaSetDbTable + "`"
	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schemas := make(stringSet)

	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}

		schemas[schema] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	log.Debug("post-migration: read %d pending schema IDs", len(schemas))
	return schemas, nil
}

func (b *postMigrationEventsBackend) ensureTableExists(tx *sql.Tx) error {
	log.Debug("post-migration: making sure table `%s` exists", schemaSetDbTable)

	const stmt = "CREATE TABLE IF NOT EXISTS `" + schemaSetDbTable + "` (`schema_id` TEXT NOT NULL)"
	_, err := tx.Exec(stmt)

	return err
}

func (b *postMigrationEventsBackend) persistSchemaID(schemaID string, tx *sql.Tx) error {
	log.Debug("post-migration: persiting schemaID '%s' into `%s`", schemaID, schemaSetDbTable)
	const stmtTemplate = "INSERT INTO `" + schemaSetDbTable + "` (`schema_id`) VALUES (?)"

	return b.execStmt(tx, stmtTemplate, schemaID)
}

func (b *postMigrationEventsBackend) deleteSchemaID(schemaID string) error {
	log.Debug("post-migration: deleting schemaID '%s' from schema set", schemaID)
	tx, err := b.db.Begin()

	if err != nil {
		return err
	}

	const stmtTemplate = "DELETE FROM `" + schemaSetDbTable + "` WHERE `schema_id` = ?"
	err = b.execStmt(tx, stmtTemplate, schemaID)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (b *postMigrationEventsBackend) execStmt(tx *sql.Tx, stmtTemplate, value string) error {
	log.Debug("post-migration: executing '%s' with '%s'", stmtTemplate, value)

	stmt, err := tx.Prepare(stmtTemplate)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(value)
	return err
}
