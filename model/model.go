// Package model implements convenience methods for
// managing indexes on top of the Store.
// See this doc for the general idea https://github.com/m3o/dev/blob/feature/storeindex/design/auto-indexes.md
// Prior art/Inspirations from github.com/gocassa/gocassa, which
// is a similar package on top an other KV store (Cassandra/gocql)
package model

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/micro/micro/v3/service/store"
)

var (
	ErrorNotFound             = errors.New("not found")
	ErrorMultipleRecordsFound = errors.New("multiple records found")
)

type OrderType string

const (
	OrderTypeUnordered = OrderType("unordered")
	OrderTypeAsc       = OrderType("ascending")
	OrderTypeDesc      = OrderType("descending")
)

const (
	queryTypeEq = "eq"
	indexTypeEq = "eq"
)

func defaultIndex() Index {
	idIndex := ByEquality("id")
	idIndex.Order.Type = OrderTypeUnordered
	return idIndex
}

type table struct {
	store store.Store
	// helps logically separate keys in a table where
	// multiple `Table`s share the same underlying
	// physical database.
	namespace string
	indexes   []Index
	options   TableOptions
}

// Table represents a place where data can be saved to and
// queried from.
type Table interface {
	// Save any object. Maintains indexes set up.
	Save(interface{}) error
	// List objects by a query. Each query requires an appropriate index
	// to exist. List throws an error if a matching index can't be found.
	List(query Query, resultSlicePointer interface{}) error
	// Same as list, but accepts pointer to non slices and
	// expects to find only one element. Throws error if not found
	// or if more than two elements are found.
	Read(query Query, resultPointer interface{}) error
	// Deletes a record. Delete only support Equals("id", value) for now.
	// @todo Delete only supports string keys for now.
	Delete(query Query) error
}

type TableOptions struct {
	Debug   bool
	IdIndex Index
}

func NewTable(store store.Store, namespace string, indexes []Index, options *TableOptions) Table {
	debug := false
	var idIndex Index
	if options != nil {
		debug = options.Debug
		idIndex = options.IdIndex
	}
	if idIndex.Type == "" {
		idIndex = defaultIndex()
	}
	return &table{
		store, namespace, indexes, TableOptions{
			Debug:   debug,
			IdIndex: idIndex,
		}}
}

type Index struct {
	FieldName string
	// Type of index, eg. equality
	Type  string
	Order Order
	// Do not allow duplicate values of this field in the index.
	// Useful for emails, usernames, post slugs etc.
	Unique bool
	// Strings for ordering will be padded to a fix length
	// Not a useful property for Querying, please ignore this at query time.
	// Number is in bytes, not string characters. Choose a sufficiently big one.
	// Consider that each character might take 4 bytes given the
	// internals of reverse ordering. So a good rule of thumbs is expected
	// characters in a string X 4
	StringOrderPadLength int
	// True = base32 encode ordered strings for easier management
	// or false = keep 4 bytes long runes that might dispaly weirdly
	Base32Encode bool
}

type Order struct {
	FieldName string
	// Ordered or unordered keys. Ordered keys are padded.
	// Default is true. This option only exists for strings, where ordering
	// comes at the cost of having rather long padded keys.
	Type OrderType
}

func (i Index) ToQuery(value interface{}) Query {
	return Query{
		Index: i,
		Value: value,
		Order: i.Order,
	}
}

func Indexes(indexes ...Index) []Index {
	return indexes
}

// ByEquality constructs an equiality index on `fieldName`
func ByEquality(fieldName string) Index {
	return Index{
		FieldName: fieldName,
		Type:      indexTypeEq,
		Order: Order{
			Type:      OrderTypeAsc,
			FieldName: fieldName,
		},
		StringOrderPadLength: 16,
		Base32Encode:         false,
	}
}

type Query struct {
	Index
	Order  Order
	Value  interface{}
	Offset int64
	Limit  int64
}

// Equals is an equality query by `fieldName`
// It filters records where `fieldName` equals to a value.
func Equals(fieldName string, value interface{}) Query {
	return Query{
		Index: Index{
			Type:      queryTypeEq,
			FieldName: fieldName,
			Order: Order{
				FieldName: fieldName,
				Type:      OrderTypeAsc,
			},
		},
		Value: value,
		Order: Order{
			FieldName: fieldName,
			Type:      OrderTypeAsc,
		},
	}
}

func (d *table) Save(instance interface{}) error {
	// @todo replace this hack with reflection
	js, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	m := map[string]interface{}{}
	de := json.NewDecoder(bytes.NewReader(js))
	de.UseNumber()
	err = de.Decode(&m)
	if err != nil {
		return err
	}

	// get the old entries so we can compare values
	// @todo consider some kind of locking (even if it's not distributed) by key here
	// to avoid 2 read-writes happening at the same time
	idQuery := d.options.IdIndex.ToQuery(m[d.options.IdIndex.FieldName])

	oldEntryList := []map[string]interface{}{}
	err = d.List(idQuery, &oldEntryList)
	if err != nil {
		return err
	}
	var oldEntry map[string]interface{}
	if len(oldEntryList) > 0 {
		oldEntry = oldEntryList[0]
	}

	// Do uniqueness checks before saving any data
	for _, index := range d.indexes {
		if !index.Unique {
			continue
		}
		res := []map[string]interface{}{}
		q := index.ToQuery(m[index.FieldName])
		err = d.List(q, &res)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			continue
		}
		if len(res) > 1 {
			return errors.New("Multiple entries found for unique index")
		}
		if res[0][d.options.IdIndex.FieldName] != m[d.options.IdIndex.FieldName] {
			return errors.New("Unique index violated")
		}
	}

	id := m[d.options.IdIndex.FieldName]
	for _, index := range append(d.indexes, d.options.IdIndex) {
		// delete non id index keys to prevent stale index values
		// ie.
		//
		//  # prefix  slug     id
		//  postByTag/hi-there/1
		//  # if slug gets changed to "hello-there" we will have two records
		//  # without removing the old stale index:
		//  postByTag/hi-there/1
		//  postByTag/hello-there/1`
		//
		// @todo this check will only work for POD types, ie no slices or maps
		// but it's not an issue as right now indexes are only supported on POD
		// types anyway
		if !indexesMatch(defaultIndex(), index) &&
			oldEntry != nil &&
			oldEntry[index.FieldName] != m[index.FieldName] {
			k := d.indexToKey(index, id, oldEntry[index.FieldName], true)
			err = d.store.Delete(k)
			if err != nil {
				return err
			}
		}
		k := d.indexToKey(index, id, m[index.FieldName], true)
		if d.options.Debug {
			fmt.Printf("Saving key '%v', value: '%v'\n", k, string(js))
		}
		err = d.store.Write(&store.Record{
			Key:   k,
			Value: js,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *table) Read(query Query, resultPointer interface{}) error {
	for _, index := range append(d.indexes, d.options.IdIndex) {
		if indexMatchesQuery(index, query) {
			k := d.queryToListKey(index, query)
			if d.options.Debug {
				fmt.Printf("Listing key '%v'\n", k)
			}
			recs, err := d.store.Read(k, store.ReadPrefix())
			if err != nil {
				return err
			}
			if len(recs) == 0 {
				return ErrorNotFound
			}
			if len(recs) > 1 {
				return ErrorMultipleRecordsFound
			}
			return json.Unmarshal(recs[0].Value, resultPointer)
		}
	}
	return fmt.Errorf("For query type '%v', field '%v' does not match any indexes", query.Type, query.FieldName)
}

func (d *table) List(query Query, resultSlicePointer interface{}) error {
	for _, index := range append(d.indexes, d.options.IdIndex) {
		if indexMatchesQuery(index, query) {
			k := d.queryToListKey(index, query)
			if d.options.Debug {
				fmt.Printf("Listing key '%v'\n", k)
			}
			recs, err := d.store.Read(k, store.ReadPrefix())
			if err != nil {
				return err
			}
			// @todo speed this up with an actual buffer
			jsBuffer := []byte("[")
			for i, rec := range recs {
				jsBuffer = append(jsBuffer, rec.Value...)
				if i < len(recs)-1 {
					jsBuffer = append(jsBuffer, []byte(",")...)
				}
			}
			jsBuffer = append(jsBuffer, []byte("]")...)
			return json.Unmarshal(jsBuffer, resultSlicePointer)
		}
	}
	return fmt.Errorf("For query type '%v', field '%v' does not match any indexes", query.Type, query.FieldName)
}

func indexMatchesQuery(i Index, q Query) bool {
	if i.FieldName == q.FieldName &&
		i.Type == q.Type &&
		i.Order.Type == q.Order.Type {
		return true
	}
	return false
}

func indexesMatch(i, j Index) bool {
	if i.FieldName == j.FieldName &&
		i.Type == j.Type &&
		i.Order.Type == j.Order.Type {
		return true
	}
	return false
}

func (d *table) queryToListKey(i Index, q Query) string {
	if q.Value == nil {
		return fmt.Sprintf("%v:%v", d.namespace, indexPrefix(i))
	}

	return d.indexToKey(i, "", q.Value, false)
}

// append id placeholder to `indexToKey` format strings if needed
func formatWrapper(appendID bool) func(format string) string {
	return func(format string) string {
		if appendID {
			return format + ":%v"
		}
		return format
	}
}

// append id value to `indexToKey` argument list if needed
func valueWrapper(appendID bool, id interface{}) func(values ...interface{}) []interface{} {
	return func(values ...interface{}) []interface{} {
		if appendID {
			return append(values, id)
		}
		return values
	}
}

// appendID true should be used when saving, false when querying
// appendID false should also be used for 'id' indexes since they already have the unique
// id. The reason id gets appended is make duplicated index keys unique.
// ie.
// # index # age # id
// users/30/1
// users/30/2
// without ids we could only have one 30 year old user in the index
func (d *table) indexToKey(i Index, id interface{}, fieldValue interface{}, appendID bool) string {
	fw := formatWrapper(appendID)
	vw := valueWrapper(appendID, id)

	switch i.Type {
	case indexTypeEq:
		fieldName := i.FieldName
		if len(fieldName) == 0 {
			fieldName = "id"
		}
		typ := reflect.TypeOf(fieldValue)
		typName := "nil"
		if typ != nil {
			typName = typ.String()
		}
		switch v := fieldValue.(type) {
		case string:
			if i.Order.Type != OrderTypeUnordered {
				return d.getOrderedStringFieldKey(i, id, v, appendID)
			}
			return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), v)...)
		case json.Number:
			// @todo some duplication going on here, see int64 and float64 cases,
			// move it out to a function
			i64, err := v.Int64()
			if err == nil {
				// int64 gets padded to 19 characters as the maximum value of an int64
				// is 9223372036854775807
				// @todo handle negative numbers
				if i.Order.Type == OrderTypeDesc {
					return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), fmt.Sprintf("%019d", math.MaxInt64-i64))...)
				}
				return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), fmt.Sprintf("%019d", i64))...)
			}
			f64, err := v.Float64()
			if err == nil {
				// @todo fix display and padding of floats
				if i.Order.Type == OrderTypeDesc {
					return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), math.MaxFloat64-f64)...)
				}
				return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), v)...)
			}
			panic("bug in code, unhandled json.Number type: " + typName + " for field " + i.FieldName)
		case int64:
			// int64 gets padded to 19 characters as the maximum value of an int64
			// is 9223372036854775807
			// @todo handle negative numbers
			if i.Order.Type == OrderTypeDesc {
				return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), fmt.Sprintf("%019d", math.MaxInt64-v))...)
			}
			return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), fmt.Sprintf("%019d", v))...)
		case float64:
			// @todo fix display and padding of floats
			if i.Order.Type == OrderTypeDesc {
				return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), math.MaxFloat64-v)...)
			}
			return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), v)...)
		case int:
			// int gets padded to the same length as int64 to gain
			// resiliency in case of model type changes.
			// This could be removed once migrations are implemented
			// so savings in space for a type reflect in savings in space in the index too.
			if i.Order.Type == OrderTypeDesc {
				return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), fmt.Sprintf("%019d", math.MaxInt32-v))...)
			}
			return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), fmt.Sprintf("%019d", v))...)
		case bool:
			return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), v)...)
		}
		panic("bug in code, unhandled type: " + typName + " for field " + i.FieldName)
	}
	return ""
}

// indexPrefix returns the first part of the keys, the namespace + index name
func indexPrefix(i Index) string {
	if i.Order.Type != OrderTypeUnordered {
		desc := ""
		if i.Order.Type == OrderTypeDesc {
			desc = "Desc"
		}
		return fmt.Sprintf("by%vOrdered%v", desc, strings.Title(i.FieldName))
	}
	return fmt.Sprintf("by%v", strings.Title(i.FieldName))
}

// pad, reverse and optionally base32 encode string keys
func (d *table) getOrderedStringFieldKey(i Index, id interface{}, fieldValue string, appendID bool) string {
	fw := formatWrapper(appendID)
	vw := valueWrapper(appendID, id)

	runes := []rune{}
	if i.Order.Type == OrderTypeDesc {
		for _, char := range fieldValue {
			runes = append(runes, utf8.MaxRune-char)
		}
	} else {
		for _, char := range fieldValue {
			runes = append(runes, char)
		}
	}

	// padding the string to a fixed length
	if len(runes) < i.StringOrderPadLength {
		pad := []rune{}
		for j := 0; j < i.StringOrderPadLength-len(runes); j++ {
			if i.Order.Type == OrderTypeDesc {
				pad = append(pad, utf8.MaxRune)
			} else {
				// space is the first non control operator char in ASCII
				// consequently in Utf8 too so we use it as the minimal character here
				// https://en.wikipedia.org/wiki/ASCII
				//
				// Displays somewhat unfortunately
				// @todo think about a better min rune value to use here.
				pad = append(pad, rune(32))
			}
		}
		runes = append(runes, pad...)
	}

	var keyPart string
	bs := []byte(string(runes))
	if i.Order.Type == OrderTypeDesc {
		if i.Base32Encode {
			// base32 hex should be order preserving
			// https://stackoverflow.com/questions/53301280/does-base64-encoding-preserve-alphabetical-ordering
			dst := make([]byte, base32.HexEncoding.EncodedLen(len(bs)))
			base32.HexEncoding.Encode(dst, bs)
			// The `=` must be replaced with a lower value than the
			// normal alphabet of the encoding since we want reverse order.
			keyPart = strings.ReplaceAll(string(dst), "=", "0")
		} else {
			keyPart = string(bs)
		}
	} else {
		keyPart = string(bs)

	}
	return fmt.Sprintf(fw("%v:%v:%v"), vw(d.namespace, indexPrefix(i), keyPart)...)
}

func (d *table) Delete(query Query) error {
	defInd := defaultIndex()
	if !indexMatchesQuery(defInd, query) {
		return errors.New("Delete query does not match default index")
	}
	results := []map[string]interface{}{}
	err := d.List(query, &results)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return errors.New("No entry found to delete")
	}
	key := d.indexToKey(defInd, results[0][d.options.IdIndex.FieldName], map[string]interface{}{
		d.options.IdIndex.FieldName: results[0][d.options.IdIndex.FieldName],
	}, true)
	fmt.Printf("Deleting key '%v'\n", key)
	return d.store.Delete(key)
}
