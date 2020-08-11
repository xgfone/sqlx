// Copyright 2020 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlx

import (
	"database/sql"
	"fmt"
)

// DB is the wrapper of the sql.DB.
type DB struct {
	*sql.DB
	Dialect
	Executor
	Interceptor
}

// Open opens a database specified by its database driver name
// and a driver-specific data source name,
func Open(driverName, dataSourceName string) (*DB, error) {
	dialect := GetDialect(driverName)
	if dialect == nil {
		return nil, fmt.Errorf("the dialect '%s' has not been registered",
			driverName)
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &DB{Dialect: dialect, DB: db}, nil
}

// CreateTable returns a SQL table builder.
func (db *DB) CreateTable(table string) *TableBuilder {
	return Table(table).SetDialect(db.Dialect).SetDB(db.DB).
		SetInterceptor(db.Interceptor).SetExecutor(db.Executor)
}

// Delete returns a DELETE SQL builder.
func (db *DB) Delete() *DeleteBuilder {
	return Delete().SetDialect(db.Dialect).SetDB(db.DB).
		SetInterceptor(db.Interceptor).SetExecutor(db.Executor)
}

// Insert returns a INSERT SQL builder.
func (db *DB) Insert() *InsertBuilder {
	return Insert().SetDialect(db.Dialect).SetDB(db.DB).
		SetInterceptor(db.Interceptor).SetExecutor(db.Executor)
}

// Select returns a SELECT SQL builder.
func (db *DB) Select(column string, alias ...string) *SelectBuilder {
	return Select(column, alias...).SetDialect(db.Dialect).SetDB(db.DB).
		SetInterceptor(db.Interceptor).SetExecutor(db.Executor)
}

// Selects is equal to db.Select(columns[0]).Select(columns[1])...
func (db *DB) Selects(columns ...string) *SelectBuilder {
	return Selects(columns...).SetDialect(db.Dialect).SetDB(db.DB).
		SetInterceptor(db.Interceptor).SetExecutor(db.Executor)
}

// SelectStruct is equal to db.Select().SelectStruct(s).
func (db *DB) SelectStruct(s interface{}) *SelectBuilder {
	return SelectStruct(s).SetDialect(db.Dialect).SetDB(db.DB).
		SetInterceptor(db.Interceptor).SetExecutor(db.Executor)
}

// Update returns a UPDATE SQL builder.
func (db *DB) Update() *UpdateBuilder {
	return Update().SetDialect(db.Dialect).SetDB(db.DB).
		SetInterceptor(db.Interceptor).SetExecutor(db.Executor)
}
