package utils

import (
	"reflect"
	"strings"
)

// JoinWithAnd joins a slice of strings with AND operator
func JoinWithAnd(clauses []string) string {
	return strings.Join(clauses, " AND ")
}

// JoinWithOr joins a slice of strings with OR operator
func JoinWithOr(clauses []string) string {
	return strings.Join(clauses, " OR ")
}
func StructArgs(v any, fields ...string) []any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	args := make([]any, len(fields))
	for i, f := range fields {
		args[i] = rv.FieldByName(f).Interface()
	}
	return args
}
