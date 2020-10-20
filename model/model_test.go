package model

import (
	"testing"

	fs "github.com/micro/micro/v3/service/store/file"
)

type User struct {
	ID  string `json:"id"`
	Age int    `json:"age"`
}

func TestBasics(t *testing.T) {
	db := NewDB(fs.NewStore(), User{}, Indexes(ByEq("age")))

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
	err = db.List(Eq("age", 12), &users)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatal(users)
	}
}
