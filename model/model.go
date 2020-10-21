// Package model implements convenience methods for
// managing indexes on top of the Store.
// See this doc for the general idea https://github.com/m3o/dev/blob/feature/storeindex/design/auto-indexes.md
// Prior art/Inspirations from github.com/gocassa/gocassa, which
// is a similar package on top an other KV store (Cassandra/gocql)
package model

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/micro/micro/v3/service/store"
)

const (
	queryTypeEq = "eq"
	indexTypeEq = "eq"
)

func defaultIndex() Index {
	idIndex := ByEquality("id")
	idIndex.Ordered = false
	return idIndex
}

type db struct {
	store   store.Store
	indexes []Index
	debug   bool
	// helps logically separate keys in a db where
	// multiple `DB`s share the same underlying
	// physical database.
	Namespace string
}

// DB represents a place where data can be saved to and
// queried from.
type DB interface {
	// Save any object. Maintains indexes set up.
	Save(interface{}) error
	// Deletes a record. Delete only support Equals("id", value) for now.
	// @todo Delete only supports string keys for now.
	Delete(query Query) error
	// List objects by a query. Each query requires an appropriate index
	// to exist. List throws an error if a matching index can't be found.
	List(query Query, resultPointer interface{}) error
}

func NewDB(store store.Store, namespace string, indexes []Index) DB {
	// filtering out the default id index in case the user passes it in
	// maybe this should do deduping instead
	filtered := []Index{}
	for _, index := range indexes {
		if !indexesMatch(index, defaultIndex()) {
			filtered = append(filtered, index)
		}
	}

	debug := false
	return &db{
		store, append([]Index{defaultIndex()}, filtered...), debug, namespace,
	}
}

type Index struct {
	FieldName string
	// Type of index, eg. equality
	Type string

	// Ordered or unordered keys. Ordered keys are padded.
	// Default is true. This option only exists for strings, where ordering
	// comes at the cost of having rather long padded keys.
	Ordered bool
	// Default order is ascending, if Desc is true it is descending
	Desc bool
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

func Indexes(indexes ...Index) []Index {
	return indexes
}

// ByEquality constructs an equiality index on `fieldName`
func ByEquality(fieldName string) Index {
	return Index{
		FieldName:            fieldName,
		Type:                 indexTypeEq,
		Ordered:              true,
		StringOrderPadLength: 16,
		Base32Encode:         false,
	}
}

type Query struct {
	Index
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
			Ordered:   true,
		},
		Value: value,
	}
}

func (d *db) Save(instance interface{}) error {
	// @todo replace this hack with reflection
	js, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	m := map[string]interface{}{}
	err = json.Unmarshal(js, &m)
	if err != nil {
		return err
	}
	id, ok := m["ID"].(string)
	if !ok || len(id) == 0 {
		id, ok = m["id"].(string)
		if !ok || len(id) == 0 {
			return fmt.Errorf("ID of objects must marshal to JSON key 'ID' or 'id'")
		}
	}
	for _, index := range d.indexes {
		k := d.indexToSaveKey(index, id, m)
		if d.debug {
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

func (d *db) List(query Query, resultSlicePointer interface{}) error {
	if len(d.indexes) == 0 {
		return errors.New("No indexes found")
	}
	for _, index := range d.indexes {
		if indexMatchesQuery(index, query) {
			k := d.queryToListKey(index, query)
			if d.debug {
				fmt.Printf("Listing key %v\n", k)
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
	if i.Type == q.Type && i.Ordered == q.Ordered && i.Desc == q.Desc {
		return true
	}
	return false
}

func indexesMatch(i, j Index) bool {
	if i.Type == j.Type && i.Ordered == j.Ordered && i.Desc == j.Desc {
		return true
	}
	return false
}

func (d *db) queryToListKey(i Index, q Query) string {
	if q.Value == nil {
		return fmt.Sprintf("%v:%v", d.Namespace, indexPrefix(i))
	}
	return fmt.Sprintf("%v:%v:%v", d.Namespace, indexPrefix(i), q.Value)
}

func (d *db) indexToSaveKey(i Index, id string, m map[string]interface{}) string {
	switch i.Type {
	case indexTypeEq:
		fieldName := i.FieldName
		if len(fieldName) == 0 {
			fieldName = "id"
		}
		switch v := m[fieldName].(type) {
		case string:
			if i.Ordered {
				return d.getOrderedStringFieldKey(i, id, v)
			}
			return fmt.Sprintf("%v:%v:%v:%v", d.Namespace, indexPrefix(i), v, id)
		case float64:
			if i.Desc {
				return fmt.Sprintf("%v:%v:%v:%v", d.Namespace, indexPrefix(i), math.MaxFloat64-v, id)
			}
			return fmt.Sprintf("%v:%v:%v:%v", d.Namespace, indexPrefix(i), v, id)
		case bool:
			return fmt.Sprintf("%v:%v:%v:%v", d.Namespace, indexPrefix(i), v, id)
		}
		return fmt.Sprintf("%v:%v:%v:%v", d.Namespace, indexPrefix(i), m[i.FieldName], id)
	}
	return ""
}

func indexPrefix(i Index) string {
	if i.Ordered {
		desc := ""
		if i.Desc {
			desc = "desc"
		}
		return fmt.Sprintf("by%vOrdered%v", desc, strings.Title(i.FieldName))
	}
	return fmt.Sprintf("by%v", strings.Title(i.FieldName))
}

// since field keys
func (d *db) getOrderedStringFieldKey(i Index, id, fieldValue string) string {
	runes := []rune{}
	if i.Desc {
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
			if i.Desc {
				pad = append(pad, utf8.MaxRune)
			} else {
				pad = append(pad, 'a')
			}
		}
		runes = append(runes, pad...)
	}

	var keyPart string
	bs := []byte(string(runes))
	if i.Desc {
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
	return fmt.Sprintf("%v:%v:%v:%v", d.Namespace, indexPrefix(i), keyPart, id)
}

func (d *db) Delete(query Query) error {
	defInd := defaultIndex()
	if !indexMatchesQuery(defInd, query) {
		return errors.New("Delete query does not match default index")
	}
	// @todo this basically limits ids to strings
	type ID struct {
		ID string `json:"id"`
	}
	results := []ID{}
	err := d.List(query, &results)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return errors.New("No entry found to delete")
	}
	key := d.indexToSaveKey(defInd, results[0].ID, map[string]interface{}{
		"id": results[0].ID,
	})
	return d.store.Delete(key)
}
