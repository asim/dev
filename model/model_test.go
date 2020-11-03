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
	Tag     string `json:"tag"`
	Updated int64  `json:"updated"`
}

func TestEqualsByID(t *testing.T) {
	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), nil, nil)

	err := table.Save(User{
		ID:  "1",
		Age: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(User{
		ID:  "2",
		Age: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	users := []User{}
	q := Equals("id", "1")
	q.Order.Type = OrderTypeUnordered
	err = table.List(q, &users)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatal(users)
	}
}

func TestRead(t *testing.T) {
	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(ByEquality("age")), nil)
	user := User{}
	err := table.Read(Equals("age", 25), &user)
	if err != ErrorNotFound {
		t.Fatal(err)
	}

	err = table.Save(User{
		ID:  "1",
		Age: 25,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = table.Read(Equals("age", 25), &user)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "1" {
		t.Fatal(user)
	}

	err = table.Save(User{
		ID:  "2",
		Age: 25,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = table.Read(Equals("age", 25), &user)
	if err != ErrorMultipleRecordsFound {
		t.Fatal(err)
	}
}

func TestEquals(t *testing.T) {
	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(ByEquality("age")), nil)

	err := table.Save(User{
		ID:  "1",
		Age: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(User{
		ID:  "2",
		Age: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(User{
		ID:  "3",
		Age: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	users := []User{}
	err = table.List(Equals("age", 12), &users)
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
		tags    []string
		reverse bool
	}
	cazes := []caze{
		{
			tags:    []string{"2", "1"},
			reverse: false,
		},
		{
			tags:    []string{"2", "1"},
			reverse: true,
		},
		{

			tags:    []string{"abcd", "abcde", "abcdf"},
			reverse: false,
		},
		{
			tags:    []string{"abcd", "abcde", "abcdf"},
			reverse: true,
		},
		{
			tags:    []string{"2", "abcd", "abcde", "abcdf", "1"},
			reverse: false,
		},
		{
			tags:    []string{"2", "abcd", "abcde", "abcdf", "1"},
			reverse: true,
		},
	}
	for _, c := range cazes {
		tagIndex := ByEquality("tag")
		if c.reverse {
			tagIndex.Order.Type = OrderTypeDesc
		}
		tagIndex.StringOrderPadLength = 12
		table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(tagIndex), nil)
		for _, key := range c.tags {
			err := table.Save(User{
				ID:  uuid.Must(uuid.NewV4()).String(),
				Tag: key,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		users := []User{}
		q := Equals("tag", nil)
		if c.reverse {
			q.Order.Type = OrderTypeDesc
		}
		err := table.List(q, &users)
		if err != nil {
			t.Fatal(err)
		}

		tags := sort.StringSlice(c.tags)
		sort.Sort(tags)
		if c.reverse {
			reverse(tags)
		}
		if len(tags) != len(users) {
			t.Fatal(tags, users)
		}
		for i, key := range tags {
			if users[i].Tag != key {
				userTags := []string{}
				for _, v := range users {
					userTags = append(userTags, v.Tag)
				}
				t.Fatalf("Should be %v, got %v, is reverse: %v", tags, userTags, c.reverse)
			}
		}
	}

}

func reverseInt(is []int) {
	last := len(is) - 1
	for i := 0; i < len(is)/2; i++ {
		is[i], is[last-i] = is[last-i], is[i]
	}
}

func TestOrderingNumbers(t *testing.T) {
	type caze struct {
		dates   []int
		reverse bool
	}
	cazes := []caze{
		{
			dates:   []int{20, 30},
			reverse: false,
		},
		{
			dates:   []int{20, 30},
			reverse: true,
		},
	}
	for _, c := range cazes {
		createdIndex := ByEquality("created")
		if c.reverse {
			createdIndex.Order.Type = OrderTypeDesc
		}
		table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(createdIndex), nil)
		for _, key := range c.dates {
			err := table.Save(User{
				ID:      uuid.Must(uuid.NewV4()).String(),
				Created: int64(key),
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		users := []User{}
		q := Equals("created", nil)
		if c.reverse {
			q.Order.Type = OrderTypeDesc
		}
		err := table.List(q, &users)
		if err != nil {
			t.Fatal(err)
		}

		dates := sort.IntSlice(c.dates)
		sort.Sort(dates)
		if c.reverse {
			reverseInt([]int(dates))
		}
		if len(users) != len(dates) {
			t.Fatalf("Expected %v, got %v", len(dates), len(users))
		}
		for i, date := range dates {
			if users[i].Created != int64(date) {
				userDates := []int{}
				for _, v := range users {
					userDates = append(userDates, int(v.Created))
				}
				t.Fatalf("Should be %v, got %v, is reverse: %v", dates, userDates, c.reverse)
			}
		}
	}

}

func TestStaleIndexRemoval(t *testing.T) {
	tagIndex := ByEquality("tag")
	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(tagIndex), nil)
	err := table.Save(User{
		ID:  "1",
		Tag: "hi-there",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(User{
		ID:  "1",
		Tag: "hello-there",
	})
	if err != nil {
		t.Fatal(err)
	}
	res := []User{}
	err = table.List(Equals("tag", nil), &res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) > 1 {
		t.Fatal(res)
	}
}

func TestUniqueIndex(t *testing.T) {
	tagIndex := ByEquality("tag")
	tagIndex.Unique = true
	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(tagIndex), nil)
	err := table.Save(User{
		ID:  "1",
		Tag: "hi-there",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(User{
		ID:  "2",
		Tag: "hello-there",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(User{
		ID:  "3",
		Tag: "hi-there",
	})
	if err == nil {
		t.Fatal("Save shoud fail with duplicate tag error because the index is unique")
	}
}

type Tag struct {
	Slug string `json:"slug"`
	Age  int    `json:"age"`
	Type string `json:"type"`
}

func TestNonIDKeys(t *testing.T) {
	slugIndex := ByEquality("slug")
	slugIndex.Order.Type = OrderTypeUnordered

	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), nil, &ModelOptions{
		IdIndex: slugIndex,
	})

	err := table.Save(Tag{
		Slug: "1",
		Age:  12,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(Tag{
		Slug: "2",
		Age:  25,
	})
	if err != nil {
		t.Fatal(err)
	}
	users := []User{}
	q := Equals("slug", "1")
	q.Order.Type = OrderTypeUnordered
	err = table.List(q, &users)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatal(users)
	}
}

// This might be an almost duplicate test, I used it to try reproduce an issue
// Leaving this here for now as we dont have enough tests anyway.
func TestListByString(t *testing.T) {
	slugIndex := ByEquality("slug")
	slugIndex.Order.Type = OrderTypeUnordered

	typeIndex := ByEquality("type")
	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(typeIndex), &ModelOptions{
		IdIndex: slugIndex,
		Debug:   false,
	})

	err := table.Save(Tag{
		Slug: "1",
		Type: "post-tag",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(Tag{
		Slug: "2",
		Type: "post-tag",
	})
	if err != nil {
		t.Fatal(err)
	}
	tags := []Tag{}
	q := Equals("type", "post-tag")
	err = table.List(q, &tags)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatal(tags)
	}
}

func TestOderByDifferentFieldThanFilterField(t *testing.T) {
	slugIndex := ByEquality("slug")
	slugIndex.Order.Type = OrderTypeUnordered

	typeIndex := ByEquality("type")
	typeIndex.Order = Order{
		Type:      OrderTypeDesc,
		FieldName: "age",
	}
	table := NewModel(fs.NewStore(), uuid.Must(uuid.NewV4()).String(), Indexes(typeIndex), &ModelOptions{
		IdIndex: slugIndex,
		Debug:   false,
	})

	err := table.Save(Tag{
		Slug: "1",
		Type: "post-tag",
		Age:  15,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(Tag{
		Slug: "2",
		Type: "post-tag",
		Age:  25,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = table.Save(Tag{
		Slug: "3",
		Type: "other-tag",
		Age:  30,
	})
	if err != nil {
		t.Fatal(err)
	}
	tags := []Tag{}
	err = table.List(typeIndex.ToQuery("post-tag"), &tags)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatal(tags)
	}
	if tags[0].Age != 25 {
		t.Fatal(tags)
	}
	if tags[1].Age != 15 {
		t.Fatal(tags)
	}

	err = table.List(typeIndex.ToQuery(nil), &tags)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 3 {
		t.Fatal(tags)
	}
}
