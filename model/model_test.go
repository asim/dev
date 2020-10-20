package model

import (
	"sort"
	"testing"

	"github.com/gofrs/uuid"
	fs "github.com/micro/micro/v3/service/store/file"
)

type User struct {
	ID      string `json:"id"`
	Age     int    `json:"age"`
	HasPet  bool   `json:"hasPet"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
}

func TestEqualsByID(t *testing.T) {
	idIndex := ByEquality("id")
	db := NewDB(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(idIndex))

	err := db.Save(User{
		ID:  "1",
		Age: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Save(User{
		ID:  "2",
		Age: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	users := []User{}
	err = db.List(Equals("id", "1"), &users)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatal(users)
	}
}

func TestEquals(t *testing.T) {
	db := NewDB(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(ByEquality("age")))

	err := db.Save(User{
		ID:  "1",
		Age: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Save(User{
		ID:  "2",
		Age: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Save(User{
		ID:  "3",
		Age: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	users := []User{}
	err = db.List(Equals("age", 12), &users)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatal(users)
	}
}

func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}

func TestOrderingStrings(t *testing.T) {
	type caze struct {
		keys    []string
		reverse bool
	}
	cazes := []caze{
		{
			keys:    []string{"1", "2"},
			reverse: false,
		},
		{
			keys:    []string{"abcd", "abcde", "abcdf"},
			reverse: true,
		},
	}
	for _, c := range cazes {
		idIndex := ByEquality("id")
		idIndex.ReverseOrder = c.reverse
		db := NewDB(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(idIndex))
		for _, key := range c.keys {
			err := db.Save(User{
				ID: key,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		users := []User{}
		q := Equals("id", nil)
		q.ReverseOrder = c.reverse
		err := db.List(q, &users)
		if err != nil {
			t.Fatal(err)
		}

		keys := sort.StringSlice(c.keys)
		if c.reverse {
			reverse(keys)
		}
		for i, key := range keys {
			if users[i].ID != key {
				t.Fatal(users)
			}
		}
	}

}
