// Package model implements convenience methods for
// managing indexes on top of the Store.
// See this doc for the general idea https://github.com/m3o/dev/blob/feature/storeindex/design/auto-indexes.md
// Prior art/Inspirations from github.com/gocassa/gocassa, which
// is a similar package on top an other KV store (Cassandra/gocql)
package model

import (
	"github.com/micro/micro/v3/service/store"
)

const (
	queryTypeEq = "eq"
)

type db struct {
	store   store.Store
	indexes []Index
	entity  interface{}
	fields  []string
}

func (d *db) Save(instance interface{}) error {
	return nil
}

func (d *db) List(resultPointer interface{}) error {

}

// DB represents a place where data can be saved to and
// queried from.
type DB interface {
	Save(interface{}) error
	List(Query) ([]interface{}, error)
}

func NewDB(store store.Store, entity interface{}, indexes []Index) DB {
	return db{
		store, indexes, entity, nil,
	}
}

type Index struct {
	FieldName string
	Type      string // eg. equality
	Ordering  bool   // ASC or DESC ordering
}

func Indexes(indexes ...Index) []Index {
	return indexes
}

// ByEq constructs an equiality index on `fieldName`
func ByEq(fieldName string) Index {

}

type Query struct {
	Type      string
	FieldName string
	Value     interface{}
}

// Eq is an equality query by `fieldName`
func Eq(fieldName string, value interface{}) Query {
	return Query{
		Type:      queryTypeEq,
		FieldName: fieldName,
		Value:     value,
	}
}
