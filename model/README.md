# Model

Package model is a convenience wrapper around what the `Store` provides.
It's main responsibility is to maintain indexes that would otherwise be maintaned by the users to enable different queries on the same data.

## Usage

The following snippets will this piece of code prepends them.

```go
import(
    model "github.com/micro/dev/model"
    fs "github.com/micro/micro/v3/service/store/file"
)

type User struct {
    ID      string `json:"id"`
    Name string    `json:"name"`
	Age     int    `json:"age"`
	HasPet  bool   `json:"hasPet"`
	Created int64  `json:"created"`
	Tag     string `json:"tag"`
	Updated int64  `json:"updated"`
}
```

## Query by field equality

For each field we want to query on we have to create an index. Index by `id` is optional.

```go
idIndex := ByEquality("id")
ageIndex := ByEquality("age")

db := model.NewDB(fs.NewStore(), "users", []model.Index{(idIndex, ageIndex})

err := db.Save(User{
    ID: "1",
    Name: "Alice",
    Age: 20,
})
if err != nil {
    // handle save error
}
err := db.Save(User{
    ID: "2",
    Name: "Jane",
    Age: 22
})
if err != nil {
    // handle save error
}

err = db.List(Equals("age", 22), &users)
if err != nil {
	// handle list error
}
fmt.Println(users)

// will print
// [{"id":"2","name":"Jane","age":22}]
```

## Ordering

Indexes by default are ordered. If we want to turn this behaviour off:

```go
ageIndex.Ordered = false
ageQuery := Equals("age", 22)
ageQuery.Ordered = false
```

### Ordering by string fields

Ordering comes for "free" when dealing with numeric or boolean fields, but it involves  in padding, inversing and order preserving base32 encoding of values to work for strings.

This can sometimes result in large keys saved, as the inverse of a small 1 byte character in a string is a 4 byte rune. Optionally adding base32 encoding on top to prevent exotic runes appearing in keys, strings blow up in size even more. If saving space is a requirement and ordering is not, ordering for strings should be turned off.

The matter is further complicated by the fact that the padding size must be specified ahead of time:

```go
nameIndex := ByEquality("name")
nameIndex.StringOrderPadLength = 10

nameQuery := Equals("age", 22)
// `StringOrderPadLength` is not needed to be specified for the query
```

To turn off base32 encoding and keep the runes:

```go
nameIndex.Base32Encode = false
```

## Design

### Restrictions

To maintain all indexes properly, all fields must be filled out when saving.
This sometimes requires a `Read, Modify, Save` pattern. In other words, partial updates will break indexes.

This could be avoided later if model does the loading itself.

## TODO

- Implement deletes
- Implement counters, for pattern inspiration see the [tags service](https://github.com/micro/services/tree/master/blog/tags)
- Test boolean indexes and its ordering
- There is a stuttering in the way `id` fields are being saved twice. ID fields since they are unique do not need `id` appended after them in the record keys.