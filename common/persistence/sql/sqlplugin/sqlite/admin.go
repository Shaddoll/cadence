// The MIT License (MIT)

// Copyright (c) 2017-2020 Uber Technologies Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package sqlite

import (
	"errors"

	"github.com/uber/cadence/common/persistence/sql/sqlplugin"
)

const (
	listTablesQuery = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
)

// CreateDatabase is not supported by sqlite
// each sqlite file is a database
func (mdb *DB) CreateDatabase(name string) error {
	return errors.New("sqlite doesn't support creating database")
}

// DropDatabase is not supported by sqlite
// each sqlite file is a database
func (mdb *DB) DropDatabase(name string) error {
	return errors.New("sqlite doesn't support dropping database")
}

// ListTables returns a list of tables in this database
func (mdb *DB) ListTables(_ string) ([]string, error) {
	var tables []string
	err := mdb.driver.SelectForSchemaQuery(sqlplugin.DbShardUndefined, &tables, listTablesQuery)
	return tables, err
}
