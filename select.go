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
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// Select is short for NewSelectBuilder.
func Select(column string, alias ...string) *SelectBuilder {
	return NewSelectBuilder(column, alias...)
}

// Selects is equal to Select(columns[0]).Select(columns[1])...
func Selects(columns ...string) *SelectBuilder {
	s := &SelectBuilder{dialect: DefaultDialect}
	return s.Selects(columns...)
}

// SelectStruct is equal to Select().SelectStruct(s, table...).
func SelectStruct(s interface{}, table ...string) *SelectBuilder {
	sb := &SelectBuilder{dialect: DefaultDialect}
	return sb.SelectStruct(s, table...)
}

// NewSelectBuilder returns a new SELECT builder.
func NewSelectBuilder(column string, alias ...string) *SelectBuilder {
	s := &SelectBuilder{dialect: DefaultDialect}
	return s.Select(column, alias...)
}

type sqlTable struct {
	Table string
	Alias string
}

type selectedColumn struct {
	Column string
	Alias  string
}

type orderby struct {
	Column string
	Order  Order
}

// Order represents the order used by ORDER BY.
type Order string

// Predefine some orders used by ORDER BY.
const (
	Asc  Order = "ASC"
	Desc Order = "DESC"
)

// JoinOn is the join on statement.
type JoinOn struct {
	Left  string
	Right string
}

// On returns a JoinOn instance.
func On(left, right string) JoinOn { return JoinOn{Left: left, Right: right} }

type joinTable struct {
	Type  string
	Table string
	Alias string
	Ons   []JoinOn
}

func (jt joinTable) Build(buf *bytes.Buffer, dialect Dialect) {
	if jt.Type != "" {
		buf.WriteByte(' ')
		buf.WriteString(jt.Type)
	}

	buf.WriteString(" JOIN ")
	buf.WriteString(dialect.Quote(jt.Table))
	if jt.Alias != "" {
		buf.WriteString(" AS ")
		buf.WriteString(dialect.Quote(jt.Alias))
	}

	if len(jt.Ons) > 0 {
		buf.WriteString(" ON ")
		for i, on := range jt.Ons {
			if i > 0 {
				buf.WriteString(" AND ")
			}
			buf.WriteString(dialect.Quote(on.Left))
			buf.WriteByte('=')
			buf.WriteString(dialect.Quote(on.Right))
		}
	}
}

// SelectBuilder is used to build the SELECT statement.
type SelectBuilder struct {
	ConditionSet

	intercept Interceptor
	executor  Executor
	dialect   Dialect
	distinct  bool
	tables    []sqlTable
	columns   []selectedColumn
	joins     []joinTable
	wheres    []Condition
	groupbys  []string
	havings   []string
	orderbys  []orderby
	limit     int64
	offset    int64
}

// Distinct marks SELECT as DISTINCT.
func (b *SelectBuilder) Distinct() *SelectBuilder {
	b.distinct = true
	return b
}

func (b *SelectBuilder) getAlias(column string, alias []string) string {
	if len(alias) != 0 && alias[0] != "" {
		return alias[0]
	} else if index := strings.IndexByte(column, '.'); index != -1 {
		column = column[index+1:]
		if index = strings.IndexByte(column, ')'); index != -1 {
			column = column[:index]
		}
		return column
	}
	return ""
}

// Select appends the selected column in SELECT.
func (b *SelectBuilder) Select(column string, alias ...string) *SelectBuilder {
	if column != "" {
		b.columns = append(b.columns, selectedColumn{column, b.getAlias(column, alias)})
	}

	return b
}

// Selects is equal to Select(columns[0]).Select(columns[1])...
func (b *SelectBuilder) Selects(columns ...string) *SelectBuilder {
	for _, c := range columns {
		b.Select(c)
	}
	return b
}

// SelectStruct reflects and extracts the fields of the struct as the selected
// columns, which supports the tag named "sql" to modify the column name.
// If the value of the tag is "-", however, the field will be ignored.
//
// If the field has the tag "table", it will be used as the table name of the field.
// If the argument "table" is given, it will override it.
func (b *SelectBuilder) SelectStruct(s interface{}, table ...string) *SelectBuilder {
	if s == nil {
		return b
	}

	v := reflect.ValueOf(s)
	switch kind := v.Kind(); kind {
	case reflect.Ptr:
		if v.IsNil() {
			return b
		}

		v = v.Elem()
		if v.Kind() != reflect.Struct {
			panic("not a pointer to struct")
		}
	case reflect.Struct:
	default:
		panic("not a struct")
	}

	var ftable string
	if len(table) != 0 {
		ftable = table[0]
	}

	vt := v.Type()
	for i, _len := 0, v.NumField(); i < _len; i++ {
		vft := vt.Field(i)
		name := vft.Name

		tag := vft.Tag.Get("sql")
		if index := strings.IndexByte(tag, ','); index > -1 {
			tag = strings.TrimSpace(tag[:index])
		}

		if tag == "-" {
			continue
		} else if tag != "" {
			name = tag
		}

		if ftable != "" {
			name = fmt.Sprintf("%s.%s", ftable, name)
		} else if table := vft.Tag.Get("table"); table != "" {
			name = fmt.Sprintf("%s.%s", table, name)
		}
		b.Select(name)
	}

	return b
}

// SelectedColumns returns the names of the selected columns.
//
// Notice: if the column has the alias, the alias will be returned instead.
func (b *SelectBuilder) SelectedColumns() []string {
	cs := make([]string, len(b.columns))
	for i, c := range b.columns {
		if c.Alias == "" {
			cs[i] = c.Column
		} else {
			cs[i] = c.Alias
		}
	}
	return cs
}

// From sets table name in SELECT.
func (b *SelectBuilder) From(table string, alias ...string) *SelectBuilder {
	b.tables = append(b.tables, sqlTable{table, b.getAlias(table, alias)})
	return b
}

// Join appends the "JOIN table ON on..." statement.
func (b *SelectBuilder) Join(table, alias string, ons ...JoinOn) *SelectBuilder {
	return b.joinTable("", table, alias, ons...)
}

// JoinLeft appends the "LEFT JOIN table ON on..." statement.
func (b *SelectBuilder) JoinLeft(table, alias string, ons ...JoinOn) *SelectBuilder {
	return b.joinTable("LEFT", table, alias, ons...)
}

// JoinLeftOuter appends the "LEFT OUTER JOIN table ON on..." statement.
func (b *SelectBuilder) JoinLeftOuter(table, alias string, ons ...JoinOn) *SelectBuilder {
	return b.joinTable("LEFT OUTER", table, alias, ons...)
}

// JoinRight appends the "RIGHT JOIN table ON on..." statement.
func (b *SelectBuilder) JoinRight(table, alias string, ons ...JoinOn) *SelectBuilder {
	return b.joinTable("RIGHT", table, alias, ons...)
}

// JoinRightOuter appends the "RIGHT OUTER JOIN table ON on..." statement.
func (b *SelectBuilder) JoinRightOuter(table, alias string, ons ...JoinOn) *SelectBuilder {
	return b.joinTable("RIGHT OUTER", table, alias, ons...)
}

// JoinFull appends the "FULL JOIN table ON on..." statement.
func (b *SelectBuilder) JoinFull(table, alias string, ons ...JoinOn) *SelectBuilder {
	return b.joinTable("FULL", table, alias, ons...)
}

// JoinFullOuter appends the "FULL OUTER JOIN table ON on..." statement.
func (b *SelectBuilder) JoinFullOuter(table, alias string, ons ...JoinOn) *SelectBuilder {
	return b.joinTable("FULL OUTER", table, alias, ons...)
}

func (b *SelectBuilder) joinTable(cmd, table, alias string, ons ...JoinOn) *SelectBuilder {
	b.joins = append(b.joins, joinTable{Type: cmd, Table: table, Alias: alias, Ons: ons})
	return b
}

// Where sets the WHERE conditions.
func (b *SelectBuilder) Where(andConditions ...Condition) *SelectBuilder {
	b.wheres = append(b.wheres, andConditions...)
	return b
}

// WhereNamedArgs is the same as Where, but uses the NamedArg as the condition.
func (b *SelectBuilder) WhereNamedArgs(args ...NamedArg) *SelectBuilder {
	for _, arg := range args {
		b.Where(b.Equal(arg.Name(), arg.Get()))
	}
	return b
}

// GroupBy resets the GROUP BY columns.
func (b *SelectBuilder) GroupBy(columns ...string) *SelectBuilder {
	b.groupbys = columns
	return b
}

// Having appends the HAVING expression.
func (b *SelectBuilder) Having(exprs ...string) *SelectBuilder {
	b.havings = append(b.havings, exprs...)
	return b
}

// OrderBy appends the column used by ORDER BY.
func (b *SelectBuilder) OrderBy(column string, order ...Order) *SelectBuilder {
	ob := orderby{Column: column}
	if len(order) > 0 {
		ob.Order = order[0]
	}
	b.orderbys = append(b.orderbys, ob)
	return b
}

// OrderByDesc appends the column used by ORDER BY DESC.
func (b *SelectBuilder) OrderByDesc(column string) *SelectBuilder {
	return b.OrderBy(column, Desc)
}

// OrderByAsc appends the column used by ORDER BY ASC.
func (b *SelectBuilder) OrderByAsc(column string) *SelectBuilder {
	return b.OrderBy(column, Asc)
}

// Limit sets the LIMIT to limit.
func (b *SelectBuilder) Limit(limit int64) *SelectBuilder {
	b.limit = limit
	return b
}

// Offset sets the OFFSET to offset.
func (b *SelectBuilder) Offset(offset int64) *SelectBuilder {
	b.offset = offset
	return b
}

// Paginate is equal to Limit(pageSize).Offset(pageNum * pageSize).
//
// Notice: pageNum starts with 0.
func (b *SelectBuilder) Paginate(pageNum, pageSize int64) *SelectBuilder {
	b.Limit(pageSize).Offset(pageNum * pageSize)
	return b
}

// Query builds the sql and executes it by *sql.DB.
func (b *SelectBuilder) Query() (Rows, error) {
	return b.QueryContext(context.Background())
}

// QueryContext builds the sql and executes it by *sql.DB.
func (b *SelectBuilder) QueryContext(ctx context.Context) (Rows, error) {
	query, args := b.Build()
	rows, err := b.executor.QueryContext(ctx, query, args...)
	return Rows{b, rows}, err
}

// QueryRow builds the sql and executes it by *sql.DB.
func (b *SelectBuilder) QueryRow() Row {
	return b.QueryRowContext(context.Background())
}

// QueryRowContext builds the sql and executes it by *sql.DB.
func (b *SelectBuilder) QueryRowContext(ctx context.Context) Row {
	query, args := b.Build()
	return Row{b, b.executor.QueryRowContext(ctx, query, args...)}
}

// SetExecutor sets the executor to exec.
func (b *SelectBuilder) SetExecutor(exec Executor) *SelectBuilder {
	b.executor = exec
	return b
}

// SetInterceptor sets the interceptor to f.
func (b *SelectBuilder) SetInterceptor(f Interceptor) *SelectBuilder {
	b.intercept = f
	return b
}

// SetDialect resets the dialect.
func (b *SelectBuilder) SetDialect(dialect Dialect) *SelectBuilder {
	b.dialect = dialect
	return b
}

// String is the same as b.Build(), except args.
func (b *SelectBuilder) String() string {
	sql, _ := b.Build()
	return sql
}

// Build builds the SELECT sql statement.
func (b *SelectBuilder) Build() (sql string, args []interface{}) {
	if len(b.tables) == 0 {
		panic("SelectBuilder: no table names")
	} else if len(b.columns) == 0 {
		panic("SelectBuilder: no selected columns")
	}

	buf := getBuffer()
	buf.WriteString("SELECT ")

	if b.distinct {
		buf.WriteString("DISTINCT ")
	}

	dialect := b.dialect
	if dialect == nil {
		dialect = DefaultDialect
	}

	// Selected Columns
	for i, column := range b.columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(dialect.Quote(column.Column))
		if column.Alias != "" {
			buf.WriteString(" AS ")
			buf.WriteString(dialect.Quote(column.Alias))
		}
	}

	// Tables
	buf.WriteString(" FROM ")
	for i, table := range b.tables {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(dialect.Quote(table.Table))
		if table.Alias != "" {
			buf.WriteString(" AS ")
			buf.WriteString(dialect.Quote(table.Alias))
		}
	}

	// Join
	for _, join := range b.joins {
		join.Build(buf, dialect)
	}

	// Where
	if _len := len(b.wheres); _len > 0 {
		expr := b.wheres[0]
		if _len > 1 {
			expr = And(b.wheres...)
		}

		buf.WriteString(" WHERE ")
		ab := NewArgsBuilder(dialect)
		buf.WriteString(expr.Build(ab))
		args = ab.Args()
	}

	// Group By & Having By
	if len(b.groupbys) > 0 {
		buf.WriteString(" GROUP BY ")
		for i, s := range b.groupbys {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(dialect.Quote(s))
		}

		if len(b.havings) > 0 {
			buf.WriteString(" HAVING ")
			for i, s := range b.havings {
				if i > 0 {
					buf.WriteString(" AND ")
				}
				buf.WriteString(s)
			}
		}
	}

	// Order By
	if len(b.orderbys) > 0 {
		buf.WriteString(" ORDER BY ")
		for i, ob := range b.orderbys {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(dialect.Quote(ob.Column))
			if ob.Order != "" {
				buf.WriteByte(' ')
				buf.WriteString(string(ob.Order))
			}
		}
	}

	// Limit & Offset
	if b.limit > 0 || b.offset > 0 {
		buf.WriteByte(' ')
		buf.WriteString(dialect.LimitOffset(b.limit, b.offset))
	}

	sql = buf.String()
	putBuffer(buf)
	return intercept(b.intercept, sql, args)
}

// Row is used to wrap sql.Row.
type Row struct {
	*SelectBuilder
	*sql.Row
}

// Rows is used to wrap sql.Rows.
type Rows struct {
	*SelectBuilder
	*sql.Rows
}

// ScanStruct is the same as Scan, but the columns are scanned into the struct
// s, which uses ScanColumnsToStruct.
func (r Row) ScanStruct(s interface{}) (err error) {
	return ScanColumnsToStruct(r.Scan, r.SelectedColumns(), s)
}

// ScanStruct is the same as Scan, but the columns are scanned into the struct
// s, which uses ScanColumnsToStruct.
func (r Rows) ScanStruct(s interface{}) (err error) {
	return ScanColumnsToStruct(r.Scan, r.SelectedColumns(), s)
}

// ScanColumnsToStruct scans the columns into the fields of the struct s,
// which supports the tag named "sql" to modify the field name. If the value
// of the tag is "-", however, the field will be ignored.
func ScanColumnsToStruct(scan func(...interface{}) error, columns []string,
	s interface{}) (err error) {
	fields := getFields(s)
	vs := make([]interface{}, len(columns))
	for i, c := range columns {
		vs[i] = fields[c].Addr().Interface()
	}
	return scan(vs...)
}

func getFields(s interface{}) map[string]reflect.Value {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr {
		panic("not a pointer to struct")
	} else if v = v.Elem(); v.Kind() != reflect.Struct {
		panic("not a pointer to struct")
	}

	vt := v.Type()
	_len := v.NumField()
	vs := make(map[string]reflect.Value, _len)
	for i := 0; i < _len; i++ {
		vft := vt.Field(i)
		name := vft.Name

		tag := vft.Tag.Get("sql")
		if index := strings.IndexByte(tag, ','); index > -1 {
			tag = strings.TrimSpace(tag[:index])
		}

		if tag == "-" {
			continue
		} else if tag != "" {
			name = tag
		}

		vs[name] = v.Field(i)
	}

	return vs
}
