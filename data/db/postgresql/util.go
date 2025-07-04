package postgresql

import (
	"fmt"
	"strings"
)

type InsertSQLOptions func(*InsertSQLFieldBuilder)

func WithIndex(index int) InsertSQLOptions {
	return func(f *InsertSQLFieldBuilder) {
		f.index = index
	}
}

func WithFieldSize(size int) InsertSQLOptions {
	return func(f *InsertSQLFieldBuilder) {
		f.fields = make([]string, 0, size)
		f.values = make([]any, 0, size)
	}
}

type InsertSQLFieldBuilder struct {
	index  int
	fields []string
	values []any
}

func NewInsertSQLFieldBuilder(opts ...InsertSQLOptions) *InsertSQLFieldBuilder {
	f := &InsertSQLFieldBuilder{
		index: 1,
	}

	for _, opt := range opts {
		opt(f)
	}

	if f.fields == nil {
		f.fields = make([]string, 0, 16)
	}

	if f.values == nil {
		f.values = make([]any, 0, 16)
	}

	return f
}

func (f *InsertSQLFieldBuilder) Append(key string, value any) {
	f.fields = append(f.fields, key)
	f.values = append(f.values, value)
}

func (f *InsertSQLFieldBuilder) Build() (colSql string, argSql string, values []any) {
	colSql = strings.Join(f.fields, ", ")

	argSqlB := strings.Builder{}

	for i := range f.fields {
		argSqlB.WriteString(fmt.Sprintf("$%d, ", f.index+i))
	}

	return colSql, argSqlB.String(), f.values
}

type UpdateSQLOptions func(*UpdateSQLFieldBuilder)

type UpdateSQLFieldBuilder struct {
	index  int
	fields []string
	values []any
}

func NewUpdateSQLFieldBuilder(index int, opts ...UpdateSQLOptions) *UpdateSQLFieldBuilder {
	f := &UpdateSQLFieldBuilder{
		index: index,
	}

	for _, opt := range opts {
		opt(f)
	}

	if f.fields == nil {
		f.fields = make([]string, 0, 16)
	}

	if f.values == nil {
		f.values = make([]any, 0, 16)
	}

	return f
}

func (f *UpdateSQLFieldBuilder) Append(key string, value any) {
	f.fields = append(f.fields, key)
	f.values = append(f.values, value)
}

func (f *UpdateSQLFieldBuilder) Build() (sql string, values []any) {
	sqlb := strings.Builder{}

	for i := range f.fields {
		sqlb.WriteString(fmt.Sprintf("%s = $%d, ", f.fields[i], f.index+i))
	}

	return sqlb.String(), f.values
}

type SelectSQLOptions func(*SelectSQLFieldBuilder)

type SelectSQLFieldBuilder struct {
	fields []string
	values []any
}

func NewSelectSQLFieldBuilder(opts ...SelectSQLOptions) *SelectSQLFieldBuilder {
	f := &SelectSQLFieldBuilder{}

	for _, opt := range opts {
		opt(f)
	}

	if f.fields == nil {
		f.fields = make([]string, 0, 16)
	}

	if f.values == nil {
		f.values = make([]any, 0, 16)
	}

	return f
}

func (f *SelectSQLFieldBuilder) Append(key string, value any) {
	f.fields = append(f.fields, key)
	f.values = append(f.values, value)
}

func (f *SelectSQLFieldBuilder) Build() (sql string, values []any) {
	sqlb := strings.Builder{}

	for i := range f.fields {
		sqlb.WriteString(fmt.Sprintf("%s, ", f.fields[i]))
	}

	return sqlb.String(), f.values
}

func AppendValueFirst(values []any, toAppend ...any) []any {
	ret := make([]any, 0, len(values)+len(toAppend))

	ret = append(ret, toAppend...)
	ret = append(ret, values...)

	return ret
}
