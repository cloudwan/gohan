// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"reflect"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/jmoiron/sqlx"

	// Import mysql lib
	_ "github.com/go-sql-driver/mysql"
	// Import go-sqlite3 lib
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const retryDB = 50
const retryDBWait = 10

const (
	configVersionColumnName   = "config_version"
	stateVersionColumnName    = "state_version"
	stateErrorColumnName      = "state_error"
	stateColumnName           = "state"
	stateMonitoringColumnName = "state_monitoring"
)

//DB is sql implementation of DB
type DB struct {
	sqlType, connectionString string
	handlers                  map[string]propertyHandler
	DB                        *sqlx.DB

	// options
	options options.Options
}

//Transaction is sql implementation of Transaction
type Transaction struct {
	transaction    *sqlx.Tx
	db             *DB
	closed         bool
	isolationLevel transaction.Type
	log            l.Logger
}

type TxInterface transaction.Transaction

func (tx *Transaction) getTxOptions(isolationLevel transaction.Type) (*sql.TxOptions, error) {
	sqlOptions := &sql.TxOptions{}
	switch isolationLevel {
	case transaction.ReadCommited:
		sqlOptions.Isolation = sql.LevelReadCommitted
	case transaction.ReadUncommitted:
		sqlOptions.Isolation = sql.LevelReadUncommitted
	case transaction.RepeatableRead:
		sqlOptions.Isolation = sql.LevelRepeatableRead
	case transaction.Serializable:
		sqlOptions.Isolation = sql.LevelSerializable
	default:
		msg := fmt.Sprintf("Unknown transaction isolation level: %s", isolationLevel)
		tx.log.Error(msg)
		return nil, fmt.Errorf(msg)
	}
	return sqlOptions, nil
}

//NewDB constructor
func NewDB(options options.Options) *DB {
	handlers := make(map[string]propertyHandler)
	//TODO(nati) dynamic configuration
	handlers["string"] = &stringHandler{}
	handlers["number"] = &numberHandler{}
	handlers["integer"] = &integerHandler{}
	handlers["object"] = &jsonHandler{}
	handlers["array"] = &jsonHandler{}
	handlers["boolean"] = &boolHandler{}
	return &DB{handlers: handlers, options: options}
}

//Options returns DB options
func (db *DB) Options() options.Options {
	return db.options
}

//propertyHandler for each propertys
type propertyHandler interface {
	encode(*schema.Property, interface{}) (interface{}, error)
	decode(*schema.Property, interface{}) (interface{}, error)
	dataType(*schema.Property) string
}

type defaultHandler struct {
}

func (handler *defaultHandler) encode(property *schema.Property, data interface{}) (interface{}, error) {
	return data, nil
}

func (handler *defaultHandler) decode(property *schema.Property, data interface{}) (interface{}, error) {
	return data, nil
}

func (handler *defaultHandler) dataType(property *schema.Property) (res string) {
	// TODO(marcin) extend types for schema. Here is pretty ugly guessing
	if property.ID == "id" || property.Relation != "" || property.Unique {
		res = "varchar(255)"
	} else {
		res = "text"
	}
	return
}

type stringHandler struct {
	defaultHandler
}

func (handler *stringHandler) encode(property *schema.Property, data interface{}) (interface{}, error) {
	switch t := data.(type) {
	case goext.MaybeString:
		if t.HasValue() {
			return t.Value, nil
		}
		return nil, nil
	}
	return data, nil
}

func (handler *stringHandler) decode(property *schema.Property, data interface{}) (interface{}, error) {
	if bytes, ok := data.([]byte); ok {
		return string(bytes), nil
	}
	return data, nil
}

type boolHandler struct{}

func (handler *boolHandler) encode(property *schema.Property, data interface{}) (interface{}, error) {
	switch t := data.(type) {
	case goext.MaybeBool:
		if t.HasValue() {
			return t.Value, nil
		}
		return nil, nil
	}
	return data, nil
}

func (handler *boolHandler) decode(property *schema.Property, data interface{}) (res interface{}, err error) {
	// different SQL drivers encode result with different type
	// so we need to do manual checks
	if data == nil {
		return nil, nil
	}
	switch t := data.(type) {
	default:
		err = fmt.Errorf("unknown type %T", t)
		return
	case []uint8: // mysql
		res, err = strconv.ParseUint(string(t), 10, 64)
		res = res.(uint64) != 0
	case int64: //apparently also mysql
		res = data.(int64) != 0
	case bool: // sqlite3
		res = data
	}
	return
}

func (handler *boolHandler) dataType(property *schema.Property) string {
	return "boolean"
}

type numberHandler struct{}

func (handler *numberHandler) encode(property *schema.Property, data interface{}) (interface{}, error) {
	return data, nil
}

func (handler *numberHandler) decode(property *schema.Property, data interface{}) (res interface{}, err error) {
	if data == nil {
		return nil, nil
	}
	switch t := data.(type) {
	default:
		return nil, fmt.Errorf("number: unknown type %T", t)

	case []uint8: // mysql
		res, _ = strconv.ParseFloat(string(t), 64)

	case float64: // sqlite3
		res = float64(t)
	case uint64: // sqlite3
		res = float64(t)
	case goext.MaybeFloat:
		if t.HasValue() {
			res = t.Value
		} else {
			res = nil
		}
	}
	return
}

func (handler *numberHandler) dataType(property *schema.Property) string {
	return "real"
}

type integerHandler struct{}

func (handler *integerHandler) encode(property *schema.Property, data interface{}) (interface{}, error) {
	switch t := data.(type) {
	case goext.MaybeInt:
		if t.HasValue() {
			return t.Value, nil
		}
		return nil, nil
	}
	return data, nil
}

func (handler *integerHandler) decode(property *schema.Property, data interface{}) (res interface{}, err error) {
	// different SQL drivers encode result with different type
	// so we need to do manual checks
	if data == nil {
		return nil, nil
	}
	switch t := data.(type) {
	default:
		return data, nil
	case []uint8: // mysql
		res, _ = strconv.ParseInt(string(t), 10, 64)
		res = int(res.(int64))
	case int64: // sqlite3
		res = int(t)
	}
	return
}

func (handler *integerHandler) dataType(property *schema.Property) string {
	return "numeric"
}

type jsonHandler struct {
}

func (handler *jsonHandler) encode(property *schema.Property, data interface{}) (interface{}, error) {
	bytes, err := json.Marshal(data)
	//TODO(nati) should handle encoding err
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (handler *jsonHandler) decode(property *schema.Property, data interface{}) (interface{}, error) {
	if bytes, ok := data.([]byte); ok {
		var ret interface{}
		err := json.Unmarshal(bytes, &ret)
		return ret, err
	}
	return data, nil
}

func (handler *jsonHandler) dataType(property *schema.Property) string {
	return "text"
}

func quote(str string) string {
	return fmt.Sprintf("`%s`", str)
}

func foreignKeyName(fromTable, fromProperty, toTable, toProperty string) string {
	name := fmt.Sprintf("%s_%s_%s_%s", fromTable, fromProperty, toTable, toProperty)
	if len(name) > 64 {
		diff := len(name) - 64
		return name[diff:]
	}
	return name
}

func (db *DB) measureTime(timeStarted time.Time, action string) {
	metrics.UpdateTimer(timeStarted, "db.%s", action)
}

func (db *DB) updateCounter(delta int64, counter string) {
	metrics.UpdateCounter(delta, "db.%s", counter)
}

//Connect connects to the db
func (db *DB) Connect(sqlType, conn string, maxOpenConn int) (err error) {
	defer db.measureTime(time.Now(), "connect")

	db.sqlType = sqlType
	db.connectionString = conn
	rawDB, err := sql.Open(db.sqlType, db.connectionString)
	if err != nil {
		return err
	}
	rawDB.SetMaxOpenConns(maxOpenConn)
	rawDB.SetMaxIdleConns(maxOpenConn)
	db.DB = sqlx.NewDb(rawDB, db.sqlType)

	if db.sqlType == "sqlite3" {
		db.DB.Exec("PRAGMA foreign_keys = ON;")
	}

	for i := 0; i < retryDB; i++ {
		err = db.DB.Ping()
		if err == nil {
			return nil
		}
		time.Sleep(retryDBWait * time.Second)
		log.Info("Retrying db connection... (%s)", err)
	}

	return fmt.Errorf("Failed to connect db")
}

// Close closes db connection
func (db *DB) Close() {
	defer db.measureTime(time.Now(), "close")
	db.DB.Close()
}

//BeginTx starts new transaction with given transaction options
func (db *DB) BeginTx(options ...transaction.Option) (tx transaction.Transaction, err error) {
	defer db.measureTime(time.Now(), "begin_tx")
	db.updateCounter(1, "begin.waiting")
	defer db.updateCounter(-1, "begin.waiting")

	params := transaction.NewTxParams(options...)

	var transx Transaction
	sqlOptions, err := transx.getTxOptions(params.IsolationLevel)
	if err != nil {
		return nil, err
	}

	rawTx, err := db.DB.BeginTxx(safeMysqlContext(params.Context), sqlOptions)
	if err != nil {
		db.updateCounter(1, "begin.failed")
		return nil, err
	}
	db.updateCounter(1, "active")
	if db.sqlType == "sqlite3" {
		rawTx.Exec("PRAGMA foreign_keys = ON;")
	}
	transx = Transaction{
		db:             db,
		transaction:    rawTx,
		closed:         false,
		isolationLevel: params.IsolationLevel,
	}
	if params.TraceID != "" {
		transx.log = l.NewLogger(l.TraceId(params.TraceID))
	} else {
		transx.log = log
	}

	if transx.isolationLevel == transaction.RepeatableRead || transx.isolationLevel == transaction.Serializable {
		tx = MakeCachedTransaction(&transx)
	} else {
		tx = &transx
	}

	transx.log.Debug("[%p] Created transaction %#v, isolation level: %s", rawTx, rawTx, transx.GetIsolationLevel())
	return
}

func (db *DB) genTableCols(s *schema.Schema, cascade bool, exclude []string) ([]string, []string, []string) {
	var cols []string
	var relations []string
	var indices []string
	schemaManager := schema.GetManager()
	for _, property := range s.Properties {
		if util.ContainsString(exclude, property.ID) {
			continue
		}
		handler := db.handlers[property.Type]
		sqlDataType := property.SQLType
		sqlDataProperties := ""
		if db.sqlType == "sqlite3" {
			sqlDataType = strings.Replace(sqlDataType, "auto_increment", "autoincrement", 1)
		}
		if sqlDataType == "" {
			sqlDataType = handler.dataType(&property)
			if property.ID == "id" {
				sqlDataProperties = " primary key"
			}
		}
		if property.ID != "id" {
			if property.Nullable {
				sqlDataProperties = " null"
			} else {
				sqlDataProperties = " not null"
			}
			if property.Unique {
				sqlDataProperties = " unique"
			}
		}

		query := "`" + property.ID + "` " + sqlDataType + sqlDataProperties

		cols = append(cols, query)
		if property.Relation != "" {
			foreignSchema, _ := schemaManager.Schema(property.Relation)
			if foreignSchema != nil {
				cascadeString := ""
				if cascade ||
					property.OnDeleteCascade ||
					(property.Relation == s.Parent && s.OnParentDeleteCascade) {
					cascadeString = "on delete cascade"
				}

				relationColumn := "id"
				if property.RelationColumn != "" {
					relationColumn = property.RelationColumn
				}

				relations = append(relations, fmt.Sprintf("constraint %s foreign key(`%s`) REFERENCES `%s`(%s) %s",
					quote(foreignKeyName(s.GetDbTableName(), property.ID, foreignSchema.GetDbTableName(), relationColumn)),
					property.ID, foreignSchema.GetDbTableName(), relationColumn, cascadeString))
			}
		}

		if property.Indexed {
			prefix := ""
			// mysql cannot index TEXT without prefix spec, while SQLite3 doesn't allow specifying key size
			if sqlDataType == "text" && db.sqlType == "mysql" {
				prefix = "(255)"
			}
			indices = append(indices, fmt.Sprintf("CREATE INDEX %s_%s_idx ON `%s`(`%s`%s);", s.GetDbTableName(), property.ID,
				s.GetDbTableName(), property.ID, prefix))
		}
	}

	for _, index := range s.Indexes {
		quotedColumns := make([]string, len(index.Columns))
		for i, column := range index.Columns {
			quotedColumns[i] = quote(column)
		}

		if db.sqlType == "sqlite3" && (index.Type == schema.Spatial || index.Type == schema.FullText) {
			log.Error("index %s won't be created since sqlite doesn't support spatial and fulltext index types", index.Name)
			continue
		}

		createIndexQuery := fmt.Sprintf(
			"CREATE %s INDEX %s ON %s(%s);",
			index.Type, index.Name, quote(s.GetDbTableName()), strings.Join(quotedColumns, ","))
		indices = append(indices, createIndexQuery)
	}
	return cols, relations, indices
}

//AlterTableDef generates alter table sql
func (db *DB) AlterTableDef(s *schema.Schema, cascade bool) (string, []string, error) {
	var existing []string
	rows, err := db.DB.Query(fmt.Sprintf("select * from `%s` limit 1;", s.GetDbTableName()))
	if err == nil {
		defer rows.Close()
		existing, err = rows.Columns()
	}

	if err != nil {
		return "", nil, err
	}

	cols, relations, indices := db.genTableCols(s, cascade, existing)
	cols = append(cols, relations...)

	if len(cols) == 0 {
		return "", nil, nil
	}
	alterTable := fmt.Sprintf("alter table`%s` add (%s);\n", s.GetDbTableName(), strings.Join(cols, ","))
	log.Debug("Altering table: " + alterTable)
	log.Debug("Altering indices: " + strings.Join(indices, ""))
	return alterTable, indices, nil
}

//GenTableDef generates create table sql
func (db *DB) GenTableDef(s *schema.Schema, cascade bool) (string, []string) {
	cols, relations, indices := db.genTableCols(s, cascade, nil)

	if s.StateVersioning() {
		cols = append(cols, quote(configVersionColumnName)+"int not null default 1")
		cols = append(cols, quote(stateVersionColumnName)+"int not null default 0")
		cols = append(cols, quote(stateErrorColumnName)+"text not null default ''")
		cols = append(cols, quote(stateColumnName)+"text not null default ''")
		cols = append(cols, quote(stateMonitoringColumnName)+"text not null default ''")
	}

	cols = append(cols, relations...)
	tableSQL := fmt.Sprintf("create table `%s` (%s);\n", s.GetDbTableName(), strings.Join(cols, ","))
	log.Debug("Creating table: " + tableSQL)
	log.Debug("Creating indices: " + strings.Join(indices, ""))
	return tableSQL, indices
}

//RegisterTable creates table in the db
func (db *DB) RegisterTable(s *schema.Schema, cascade, migrate bool) error {
	if s.IsAbstract() {
		return nil
	}
	tableDef, indices, err := db.AlterTableDef(s, cascade)
	if !migrate {
		if tableDef != "" || (indices != nil && len(indices) > 0) {
			return fmt.Errorf("needs migration, run \"gohan migrate\"")
		}
	}
	if err != nil {
		tableDef, indices = db.GenTableDef(s, cascade)
	}
	if tableDef != "" {
		if _, err = db.DB.Exec(tableDef); err != nil {
			return errors.Errorf("error when exec table stmt: '%s': %s", tableDef, err)
		}
	}
	for _, indexSQL := range indices {
		if _, err = db.DB.Exec(indexSQL); err != nil {
			return errors.Errorf("error when exec index stmt: '%s': %s", indexSQL, err)
		}
	}
	return err
}

//DropTable drop table definition
func (db *DB) DropTable(s *schema.Schema) error {
	if s.IsAbstract() {
		return nil
	}
	sql := fmt.Sprintf("drop table if exists %s\n", quote(s.GetDbTableName()))
	_, err := db.DB.Exec(sql)
	return err
}

func (tx *Transaction) logQuery(sql string, args ...interface{}) {
	sqlFormat := strings.Replace(sql, "%", "%%", -1)
	sqlFormat = strings.Replace(sqlFormat, "?", "%s", -1)
	query := fmt.Sprintf(sqlFormat, args...)
	tx.log.Debug("[%p] Executing SQL query '%s'", tx.transaction, query)
}

func (tx *Transaction) measureTime(timeStarted time.Time, schemaId, action string) {
	metrics.UpdateTimer(timeStarted, "tx.%s.%s", schemaId, action)
}

// Exec executes sql in transaction
func (tx *Transaction) Exec(ctx context.Context, sql string, args ...interface{}) error {
	defer tx.measureTime(time.Now(), "unknown_schema", "exec")
	return tx.exec(ctx, sql, args...)
}

func (tx *Transaction) exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := tx.execWithResult(ctx, sql, args...)
	return err
}

func (tx *Transaction) execWithResult(ctx context.Context, sql string, args ...interface{}) (transaction.Result, error) {
	tx.logQuery(sql, args...)
	return tx.transaction.ExecContext(safeMysqlContext(ctx), sql, args...)
}

//Create create resource in the db
func (tx *Transaction) Create(ctx context.Context, resource *schema.Resource) (transaction.Result, error) {
	defer tx.measureTime(time.Now(), resource.Schema().ID, "create")

	var cols []string
	var values []interface{}
	db := tx.db
	s := resource.Schema()
	data := resource.Data()
	q := sq.Insert(quote(s.GetDbTableName()))
	for _, attr := range s.Properties {
		//TODO(nati) support optional value
		if _, ok := data[attr.ID]; ok {
			handler := db.handler(&attr)
			cols = append(cols, quote(attr.ID))
			encoded, err := handler.encode(&attr, data[attr.ID])
			if err != nil {
				return nil, fmt.Errorf("SQL Create encoding error: %s", err)
			}
			values = append(values, encoded)
		}
	}
	q = q.Columns(cols...).Values(values...)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	return tx.execWithResult(ctx, sql, args...)
}

func (tx *Transaction) updateQuery(resource *schema.Resource) (sq.UpdateBuilder, error) {
	s := resource.Schema()
	db := tx.db
	data := resource.Data()
	q := sq.Update(quote(s.GetDbTableName()))
	for _, attr := range s.Properties {
		//TODO(nati) support optional value
		if _, ok := data[attr.ID]; ok {
			handler := db.handler(&attr)
			encoded, err := handler.encode(&attr, data[attr.ID])
			if err != nil {
				return q, fmt.Errorf("SQL Update encoding error: %s", err)
			}
			q = q.Set(quote(attr.ID), encoded)
		}
	}
	if s.Parent != "" {
		q = q.Set(s.ParentSchemaPropertyID(), resource.ParentID())
	}
	return q, nil
}

//Update update resource in the db
func (tx *Transaction) Update(ctx context.Context, resource *schema.Resource) error {
	defer tx.measureTime(time.Now(), resource.Schema().ID, "update")

	q, err := tx.updateQuery(resource)
	if err != nil {
		return err
	}
	sql, args, err := q.ToSql()
	if err != nil {
		return err
	}
	if resource.Schema().StateVersioning() {
		sql += ", `" + configVersionColumnName + "` = `" + configVersionColumnName + "` + 1"
	}
	sql += " WHERE id = ?"
	args = append(args, resource.ID())
	return tx.exec(ctx, sql, args...)
}

//StateUpdate update resource state
func (tx *Transaction) StateUpdate(ctx context.Context, resource *schema.Resource, state *transaction.ResourceState) error {
	defer tx.measureTime(time.Now(), resource.Schema().ID, "state_update")

	q, err := tx.updateQuery(resource)
	if err != nil {
		return err
	}
	if resource.Schema().StateVersioning() && state != nil {
		q = q.Set(quote(stateVersionColumnName), state.StateVersion)
		q = q.Set(quote(stateErrorColumnName), state.Error)
		q = q.Set(quote(stateColumnName), state.State)
		q = q.Set(quote(stateMonitoringColumnName), state.Monitoring)
	}
	q = q.Where(sq.Eq{"id": resource.ID()})
	sql, args, err := q.ToSql()
	if err != nil {
		return err
	}
	return tx.exec(ctx, sql, args...)
}

//Delete delete resource from db
func (tx *Transaction) Delete(ctx context.Context, s *schema.Schema, resourceID interface{}) error {
	defer tx.measureTime(time.Now(), s.ID, "delete")

	sql, args, err := sq.Delete(quote(s.GetDbTableName())).Where(sq.Eq{"id": resourceID}).ToSql()
	if err != nil {
		return err
	}
	return tx.exec(ctx, sql, args...)
}

func (db *DB) handler(property *schema.Property) propertyHandler {
	handler, ok := db.handlers[property.Type]
	if ok {
		return handler
	}
	return &defaultHandler{}
}

func makeColumnID(tableName string, property schema.Property) string {
	return fmt.Sprintf("%s__%s", tableName, property.ID)
}

func makeColumn(tableName string, property schema.Property) string {
	return fmt.Sprintf("%s.%s", tableName, quote(property.ID))
}

func makeAliasTableName(tableName string, property schema.Property) string {
	return fmt.Sprintf("%s__%s", tableName, property.RelationProperty)
}

// MakeColumns generates an array that has Gohan style column names
func MakeColumns(s *schema.Schema, tableName string, fields []string, join bool) []string {
	manager := schema.GetManager()

	var include map[string]bool
	if fields != nil {
		include = make(map[string]bool)
		for _, f := range fields {
			include[f] = true
		}
	}

	var cols []string
	for _, property := range s.Properties {
		if property.RelationProperty != "" && join {
			relatedSchema, ok := manager.Schema(property.Relation)
			if !ok {
				panic(fmt.Sprintf("missing schema %s", property.Relation))
			}
			aliasTableName := makeAliasTableName(tableName, property)
			cols = append(cols, MakeColumns(relatedSchema, aliasTableName, fields, true)...)
		}

		if include != nil && !include[normField(property.ID, s.ID)] {
			continue
		}

		cols = append(cols, makeColumn(tableName, property)+" as "+quote(makeColumnID(tableName, property)))
	}
	return cols
}

func makeStateColumns(s *schema.Schema) (cols []string) {
	dbTableName := s.GetDbTableName()
	cols = append(cols, dbTableName+"."+configVersionColumnName+" as "+quote(configVersionColumnName))
	cols = append(cols, dbTableName+"."+stateVersionColumnName+" as "+quote(stateVersionColumnName))
	cols = append(cols, dbTableName+"."+stateErrorColumnName+" as "+quote(stateErrorColumnName))
	cols = append(cols, dbTableName+"."+stateColumnName+" as "+quote(stateColumnName))
	cols = append(cols, dbTableName+"."+stateMonitoringColumnName+" as "+quote(stateMonitoringColumnName))
	return cols
}

func makeJoin(s *schema.Schema, tableName string, q sq.SelectBuilder) sq.SelectBuilder {
	manager := schema.GetManager()
	for _, property := range s.Properties {
		if property.RelationProperty == "" {
			continue
		}
		relatedSchema, _ := manager.Schema(property.Relation)
		aliasTableName := makeAliasTableName(tableName, property)
		q = q.LeftJoin(
			fmt.Sprintf("%s as %s on %s.%s = %s.id", quote(relatedSchema.GetDbTableName()), quote(aliasTableName),
				quote(tableName), quote(property.ID), quote(aliasTableName)))
		q = makeJoin(relatedSchema, aliasTableName, q)
	}
	return q
}

func decodeState(data map[string]interface{}, state *transaction.ResourceState) error {
	var ok bool
	state.ConfigVersion, ok = data[configVersionColumnName].(int64)
	if !ok {
		return fmt.Errorf("Wrong state column %s returned from query", configVersionColumnName)
	}
	state.StateVersion, ok = data[stateVersionColumnName].(int64)
	if !ok {
		return fmt.Errorf("Wrong state column %s returned from query", stateVersionColumnName)
	}
	stateError, ok := data[stateErrorColumnName].([]byte)
	if !ok {
		return fmt.Errorf("Wrong state column %s returned from query", stateErrorColumnName)
	}
	state.Error = string(stateError)
	stateState, ok := data[stateColumnName].([]byte)
	if !ok {
		return fmt.Errorf("Wrong state column %s returned from query", stateColumnName)
	}
	state.State = string(stateState)
	stateMonitoring, ok := data[stateMonitoringColumnName].([]byte)
	if !ok {
		return fmt.Errorf("Wrong state column %s returned from query", stateMonitoringColumnName)
	}
	state.Monitoring = string(stateMonitoring)
	return nil
}

//normFields runs normFields on all the fields.
func normFields(fields []string, s *schema.Schema) []string {
	if fields != nil {
		for i, f := range fields {
			fields[i] = normField(f, s.ID)
		}
	}
	return fields
}

//normField returns field prefixed with schema ID.
func normField(field, schemaID string) string {
	if strings.Contains(field, ".") {
		return field
	}
	return fmt.Sprintf("%s.%s", schemaID, field)
}

type selectContext struct {
	schema    *schema.Schema
	filter    transaction.Filter
	fields    []string
	join      bool
	paginator *pagination.Paginator
}

func buildSelect(sc *selectContext) (string, []interface{}, error) {
	t := sc.schema.GetDbTableName()

	cols := MakeColumns(sc.schema, t, sc.fields, sc.join)
	q := sq.Select(cols...).From(quote(t))
	q, err := AddFilterToQuery(sc.schema, q, sc.filter, sc.join)
	if err != nil {
		return "", nil, err
	}
	if sc.paginator != nil {
		if sc.paginator.Key != "" {
			property, err := sc.schema.GetPropertyByID(sc.paginator.Key)
			if err == nil {
				q = q.OrderBy(makeColumn(t, *property) + " " + sc.paginator.Order)
			}
		}

		if sc.paginator.Limit != math.MaxUint64 {
			q = q.Limit(uint64(sc.paginator.Limit))
		}
		if sc.paginator.Offset > 0 {
			q = q.Offset(sc.paginator.Offset)
		}
	}
	if sc.join {
		q = makeJoin(sc.schema, t, q)
	}
	return q.ToSql()
}

func (tx *Transaction) executeSelect(ctx context.Context, sc *selectContext, sql string, args []interface{}) (list []*schema.Resource, total uint64, err error) {
	tx.logQuery(sql, args...)
	rows, err := tx.transaction.QueryxContext(safeMysqlContext(ctx), sql, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	list, err = tx.decodeRows(sc.schema, rows, list, sc.fields != nil, sc.join)
	if err != nil {
		return nil, 0, err
	}
	total, err = tx.Count(ctx, sc.schema, sc.filter)
	return
}

//List resources in the db
func (tx *Transaction) List(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator) (list []*schema.Resource, total uint64, err error) {
	defer tx.measureTime(time.Now(), s.ID, "list")

	sc := listContextHelper(s, filter, options, pg)

	sql, args, err := buildSelect(sc)
	if err != nil {
		return nil, 0, err
	}

	return tx.executeSelect(ctx, sc, sql, args)
}

func listContextHelper(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator) *selectContext {
	sc := &selectContext{
		schema:    s,
		filter:    filter,
		join:      true,
		paginator: pg,
	}
	if options != nil {
		sc.fields = normFields(options.Fields, s)
		sc.join = options.Details
	}
	return sc
}

func shouldJoin(policy schema.LockPolicy) bool {
	switch policy {
	case schema.LockRelatedResources:
		return true
	case schema.SkipRelatedResources:
		return false
	default:
		log.Fatalf("Unknown lock policy %+v", policy)
		panic("Unexpected locking policy")
	}
}

// LockList locks resources in the db
func (tx *Transaction) LockList(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator, lockPolicy schema.LockPolicy) (list []*schema.Resource, total uint64, err error) {
	defer tx.measureTime(time.Now(), s.ID, "lock_list")

	sc := lockListContextHelper(s, filter, options, pg, lockPolicy)

	sql, args, err := buildSelect(sc)
	if err != nil {
		return nil, 0, err
	}

	if tx.db.sqlType == "mysql" {
		sql += " FOR UPDATE"
	}

	// update join for recursive
	if options != nil {
		sc.join = options.Details
	} else {
		sc.join = true
	}

	return tx.executeSelect(ctx, sc, sql, args)
}

func lockListContextHelper(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator, lockPolicy schema.LockPolicy) *selectContext {
	policyJoin := shouldJoin(lockPolicy)
	sc := &selectContext{
		schema:    s,
		filter:    filter,
		join:      policyJoin,
		paginator: pg,
	}
	if options != nil {
		sc.fields = normFields(options.Fields, s)
		sc.join = policyJoin && options.Details
	}
	return sc
}

// Query with raw sql string
func (tx *Transaction) Query(ctx context.Context, s *schema.Schema, query string, arguments []interface{}) (list []*schema.Resource, err error) {
	defer tx.measureTime(time.Now(), s.ID, "query")

	tx.logQuery(query, arguments...)
	rows, err := tx.transaction.QueryxContext(safeMysqlContext(ctx), query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("Failed to run query: %s, %s", query, err)
	}

	defer rows.Close()
	list, err = tx.decodeRows(s, rows, list, false, false)
	if err != nil {
		return nil, err
	}

	return
}

func (tx *Transaction) decodeRows(s *schema.Schema, rows *sqlx.Rows, list []*schema.Resource, skipNil, recursive bool) ([]*schema.Resource, error) {
	for rows.Next() {
		data := map[string]interface{}{}
		rows.MapScan(data)

		resourceData := tx.decode(s, s.GetDbTableName(), skipNil, recursive, data)
		resource := schema.NewResource(s, resourceData)
		list = append(list, resource)
	}

	return list, nil
}

func (tx *Transaction) decode(s *schema.Schema, tableName string, skipNil, recursive bool, data map[string]interface{}) map[string]interface{} {
	resourceData := map[string]interface{}{}

	manager := schema.GetManager()
	db := tx.db
	for _, property := range s.Properties {
		handler := db.handler(&property)
		value := data[makeColumnID(tableName, property)]
		if value != nil || (property.Nullable && !skipNil) {
			decoded, err := handler.decode(&property, value)
			if err != nil {
				tx.log.Error(fmt.Sprintf("SQL List decoding error: %s", err))
			}
			resourceData[property.ID] = decoded
		}
		if property.RelationProperty != "" && recursive {
			relatedSchema, _ := manager.Schema(property.Relation)
			aliasTableName := makeAliasTableName(tableName, property)
			relatedResourceData := tx.decode(relatedSchema, aliasTableName, skipNil, recursive, data)
			if len(relatedResourceData) > 0 || !skipNil {
				resourceData[property.RelationProperty] = relatedResourceData
			}
		}
	}

	return resourceData
}

//Count count all matching resources in the db
func (tx *Transaction) Count(ctx context.Context, s *schema.Schema, filter transaction.Filter) (res uint64, err error) {
	defer tx.measureTime(time.Now(), s.ID, "count")

	q := sq.Select("Count(id) as count").From(quote(s.GetDbTableName()))
	//Filter get already tested
	q, _ = AddFilterToQuery(s, q, filter, false)
	sql, args, err := q.ToSql()
	if err != nil {
		return
	}
	result := map[string]interface{}{}
	err = tx.transaction.QueryRowxContext(safeMysqlContext(ctx), sql, args...).MapScan(result)
	if err != nil {
		return
	}
	count, _ := result["count"]
	decoder := &integerHandler{}
	decoded, decodeErr := decoder.decode(nil, count)
	if decodeErr != nil {
		err = fmt.Errorf("SQL List decoding error: %s", decodeErr)
		return
	}
	res = uint64(decoded.(int))
	return
}

//Fetch resources by ID in the db
func (tx *Transaction) Fetch(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions) (*schema.Resource, error) {
	defer tx.measureTime(time.Now(), s.ID, "fetch")

	list, _, err := tx.List(ctx, s, filter, options, nil)
	return fetchContextHelper(list, err, filter)
}

func fetchContextHelper(list []*schema.Resource, err error, filter transaction.Filter) (*schema.Resource, error) {
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch %s: %s", filter, err)
	}
	if len(list) < 1 {
		return nil, transaction.ErrResourceNotFound
	}
	return list[0], nil
}

// LockFetch fetches & locks a resource
func (tx *Transaction) LockFetch(ctx context.Context, s *schema.Schema, filter transaction.Filter, lockPolicy schema.LockPolicy, options *transaction.ViewOptions) (*schema.Resource, error) {
	defer tx.measureTime(time.Now(), s.ID, "lock_fetch")

	list, _, err := tx.LockList(ctx, s, filter, nil, nil, lockPolicy)
	return lockFetchContextHelper(err, list, filter)
}

func lockFetchContextHelper(err error, list []*schema.Resource, filter transaction.Filter) (*schema.Resource, error) {
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch and lock %s: %s", filter, err)
	}
	if len(list) < 1 {
		return nil, transaction.ErrResourceNotFound
	}
	return list[0], nil
}

//StateFetch fetches the state of the specified resource
func (tx *Transaction) StateFetch(ctx context.Context, s *schema.Schema, filter transaction.Filter) (state transaction.ResourceState, err error) {
	defer tx.measureTime(time.Now(), s.ID, "state_fetch")

	if !s.StateVersioning() {
		err = fmt.Errorf("Schema %s does not support state versioning", s.ID)
		return
	}
	cols := makeStateColumns(s)
	q := sq.Select(cols...).From(quote(s.GetDbTableName()))
	q, _ = AddFilterToQuery(s, q, filter, true)
	sql, args, err := q.ToSql()
	if err != nil {
		return
	}
	tx.logQuery(sql, args...)
	rows, err := tx.transaction.QueryxContext(safeMysqlContext(ctx), sql, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	if !rows.Next() {
		err = transaction.ErrResourceNotFound
		return
	}
	data := map[string]interface{}{}
	rows.MapScan(data)
	err = decodeState(data, &state)
	return
}

//RawTransaction returns raw transaction
func (tx *Transaction) RawTransaction() *sqlx.Tx {
	return tx.transaction
}

//Commit commits transaction
func (tx *Transaction) Commit() error {
	defer tx.db.measureTime(time.Now(), "commit")
	defer tx.db.updateCounter(-1, "active")

	tx.log.Debug("[%p] Committing transaction %#v", tx.transaction, tx)
	err := tx.transaction.Commit()
	if err != nil {
		tx.log.Error("[%p] Commit %#v failed: %s", tx.transaction, tx, err)
		tx.db.updateCounter(1, "commit.failed")
		return err
	}
	tx.closed = true
	return nil
}

//Close closes connection
func (tx *Transaction) Close() error {
	defer tx.db.measureTime(time.Now(), "rollback")

	//Rollback if it isn't committed yet
	tx.log.Debug("[%p] Closing transaction %#v", tx.transaction, tx)
	var err error
	if !tx.closed {
		defer tx.db.updateCounter(-1, "active")
		tx.log.Debug("[%p] Rolling back %#v", tx.transaction, tx)
		err = tx.transaction.Rollback()
		if err != nil {
			tx.log.Error("[%p] Rolling back %#v failed: %s", tx.transaction, tx, err)
			tx.db.updateCounter(1, "rollback.failed")
			return err
		}
		tx.closed = true
	}
	return nil
}

//Closed returns whether the transaction is closed
func (tx *Transaction) Closed() bool {
	return tx.closed
}

// GetIsolationLevel returns tx isolation level
func (tx *Transaction) GetIsolationLevel() transaction.Type {
	return tx.isolationLevel
}

const (
	OrCondition   = "__or__"
	AndCondition  = "__and__"
	BoolCondition = "__bool__"
)

func AddFilterToQuery(s *schema.Schema, q sq.SelectBuilder, filter map[string]interface{}, join bool) (sq.SelectBuilder, error) {
	if filter == nil {
		return q, nil
	}
	for key, value := range filter {
		if key == OrCondition {
			orFilter, err := addOrToQuery(s, q, value, join)
			if err != nil {
				return q, err
			}
			q = q.Where(orFilter)
			continue
		} else if key == AndCondition {
			andFilter, err := addAndToQuery(s, q, value, join)
			if err != nil {
				return q, err
			}
			q = q.Where(andFilter)
			continue
		} else if b, ok := filter[BoolCondition]; ok {
			if b.(bool) {
				q = q.Where("(1=1)")
			} else {
				q = q.Where("(1=0)")
			}
			continue
		}
		property, err := s.GetPropertyByID(key)

		if err != nil {
			return q, err
		}

		var column string
		if join {
			column = makeColumn(s.GetDbTableName(), *property)
		} else {
			column = quote(key)
		}

		queryValues, ok := value.([]string)
		if ok && property.Type == "boolean" {
			v := make([]bool, len(queryValues))
			for i, j := range queryValues {
				v[i], _ = strconv.ParseBool(j)
			}
			q = q.Where(sq.Eq{column: v})
		} else if reflect.TypeOf(value).String() == "string" || reflect.TypeOf(value).String() == "bool"{
			q = q.Where(sq.Like{column: value})
		}  else{
			q = q.Where(sq.Like{column: string("%"+value.([]string)[0]+"%")})
		}
	}
	return q, nil
}

func addOrToQuery(s *schema.Schema, q sq.SelectBuilder, filter interface{}, join bool) (sq.Or, error) {
	return addToFilter(s, q, filter, join, sq.Or{})
}

func addAndToQuery(s *schema.Schema, q sq.SelectBuilder, filter interface{}, join bool) (sq.And, error) {
	return addToFilter(s, q, filter, join, sq.And{})
}

func addToFilter(s *schema.Schema, q sq.SelectBuilder, filter interface{}, join bool, sqlizer []sq.Sqlizer) ([]sq.Sqlizer, error) {
	filters := filter.([]map[string]interface{})
	for _, filter := range filters {
		if match, ok := filter[OrCondition]; ok {
			res, err := addOrToQuery(s, q, match, join)
			if err != nil {
				return nil, err
			}
			sqlizer = append(sqlizer, res)
		} else if match, ok := filter[AndCondition]; ok {
			res, err := addAndToQuery(s, q, match, join)
			if err != nil {
				return nil, err
			}
			sqlizer = append(sqlizer, res)
		} else if b, ok := filter[BoolCondition]; ok {
			if b.(bool) {
				sqlizer = append(sqlizer, sq.Expr("(1=1)"))
			} else {
				sqlizer = append(sqlizer, sq.Expr("(1=0)"))
			}
		} else {
			key := filter["property"].(string)
			property, err := s.GetPropertyByID(key)
			if err != nil {
				return nil, err
			}

			var column string
			if join {
				column = makeColumn(s.GetDbTableName(), *property)
			} else {
				column = quote(key)
			}

			// TODO: add other operators
			value := filter["value"]
			switch filter["type"] {
			case "eq":
				sqlizer = append(sqlizer, sq.Eq{column: value})
			case "neq":
				sqlizer = append(sqlizer, sq.NotEq{column: value})
			default:
				panic("type has to be one of [eq, neq]")
			}
		}
	}
	return sqlizer, nil
}

//SetMaxOpenConns limit maximum connections
func (db *DB) SetMaxOpenConns(maxIdleConns int) {
	// db.DB.SetMaxOpenConns(maxIdleConns)
	// db.DB.SetMaxIdleConns(maxIdleConns)
}

// Mysql driver does not support graceful cancellation via context.Cancel()
// This function (and its usages) should be removed when (if) this defect got fixed:
// https://github.com/go-sql-driver/mysql/issues/731
func safeMysqlContext(_ context.Context) context.Context {
	return context.Background()
}
