# go-backend-db

Types and a typed Go client for [go-backend-db-handler](https://github.com/ahmedbaig/go-backend-db-handler), the service that owns database access. Services never open database connections themselves; they describe the query and the handler runs it.

```sh
go get github.com/ahmedbaig/go-backend-db
```

## Typed services

```go
type Post struct {
	ID        int64  `json:"id"`
	AuthorID  int64  `json:"authorId"`
	Title     string `json:"title"`
	Published bool   `json:"published"`
}

client := db.NewClient("http://db-handler:8090",
	db.WithHTTPClient(httpx.NewClient(nil)),
	db.WithHeader("X-Api-Key", key))
posts := db.NewService[Post](client, "blog", "posts")

list, err := posts.FindMany(ctx, db.Params{
	Where:   db.W().Eq("published", true).Field("title", db.Contains("go")),
	OrderBy: []db.Order{{Field: "id", Direction: db.Desc}},
	Take:    db.Take(20),
	Include: []string{"author"},
})

p, err := posts.FindOne(ctx, db.W().Eq("id", 7))
if errors.Is(err, db.ErrNotFound) { ... }
```

| Method | Returns | Notes |
|---|---|---|
| `FindOne(where)` | `T` | `ErrNotFound` if none, `ErrConflict` if the filter matches more than one |
| `FindFirst(params)` | `T` | zero value if none |
| `FindMany(params)` | `[]T` | always pass `Take`; the handler caps it |
| `Count(where)` | `int64` | |
| `Create(v)` | `T` | with generated ID and defaults |
| `CreateMany(vs)` | `int64` | inserted in batches |
| `Update(where, changes)` | `T` | exactly one record, returns it after the update |
| `UpdateMany(where, changes)` | `int64` | rows changed |
| `Upsert(conflict, create, update)` | `T` | insert, or update on a unique conflict |
| `Delete(where)` | `T` | exactly one record, returns what was deleted |
| `DeleteMany(where)` | `int64` | rows deleted |

Writes other than create always need a non-empty `where`.

## Filters

```json
{
  "email": "a@example.com",
  "age": { "gte": 18, "lt": 65 },
  "deletedAt": null,
  "OR": [{ "role": { "in": ["admin", "owner"] } }, { "name": { "startsWith": "A" } }],
  "NOT": { "status": "banned" }
}
```

Operators: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `notIn`, `contains`, `startsWith`, `endsWith`, `isNull`. A bare value means `eq`, `null` means `isNull`. Build the same thing in Go with `db.W().Eq(...)`, `.Field(name, db.Gte(...))`, `db.AndOf`, `db.OrOf`, `db.NotOf`, `db.In`, `db.Between`.

## Raw protocol

`client.Query(ctx, db.Query{Database, Instance, Service, Method, Params})`, `client.Transaction(ctx, db.Transaction{...})` (all queries in one database transaction, all or nothing) and `client.Capabilities(ctx)` (every database, service and field the handler knows). Errors come back as `*db.RemoteError` with the HTTP status, error code and request ID, and match `ErrNotFound`, `ErrConflict` and `ErrValidation` with `errors.Is`.

`Instance` picks one of several databases with the same schema (for example one per region); set it per client with `db.WithInstance("eu")`.

## License

MIT
