package db

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestWhereJSONRoundTrip(t *testing.T) {
	in := `{"email":"a@b.c","age":{"gte":18,"lt":65},"deletedAt":null,"OR":[{"role":{"in":["admin","owner"]}},{"name":{"startsWith":"A"}}],"NOT":{"status":"banned"}}`
	var w Where
	if err := json.Unmarshal([]byte(in), &w); err != nil {
		t.Fatal(err)
	}
	if w.Fields["email"][OpEq] != "a@b.c" || w.Fields["age"][OpGte] != float64(18) || w.Fields["deletedAt"][OpIsNull] != true {
		t.Fatalf("fields %+v", w.Fields)
	}
	if len(w.Or) != 2 || w.Not == nil || w.Not.Fields["status"][OpEq] != "banned" {
		t.Fatalf("logic %+v", w)
	}
	b, _ := json.Marshal(w)
	var again Where
	_ = json.Unmarshal(b, &again)
	if !reflect.DeepEqual(w, again) {
		t.Fatalf("round trip\n%+v\n%+v", w, again)
	}
}

func TestEqualityObjectIsNotOperator(t *testing.T) {
	var w Where
	if err := json.Unmarshal([]byte(`{"meta":{"colour":"red"}}`), &w); err == nil {
		t.Fatal("unknown operator object should be rejected")
	}
}

func TestBuilders(t *testing.T) {
	w := W().Eq("shopId", "s1").Field("price", Between(10, 20))
	w = AndOf(w, OrOf(W().Field("tag", In("a", "b")), NotOf(W().Field("name", Contains("x")))))
	b, err := json.Marshal(Params{Where: w, Take: Take(5)})
	if err != nil {
		t.Fatal(err)
	}
	var p Params
	if err := json.Unmarshal(b, &p); err != nil || *p.Take != 5 || len(p.Where.And) != 2 {
		t.Fatalf("%s %v", b, err)
	}
	empty, _ := json.Marshal(Params{})
	if string(empty) != `{}` {
		t.Fatalf("empty params = %s", empty)
	}
}

type user struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestServiceClient(t *testing.T) {
	var got Query
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathQuery || r.Header.Get("X-Api-Key") != "k" {
			w.WriteHeader(400)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		switch got.Method {
		case FindMany:
			_, _ = w.Write([]byte(`{"data":[{"id":1,"name":"a"},{"id":2,"name":"b"}]}`))
		case Count:
			_, _ = w.Write([]byte(`{"count":2}`))
		case FindOne:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"requestId":"r1","errors":[{"code":"E404-1","message":"not found"}]}`))
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithInstance("eu"), WithHeader("X-Api-Key", "k"))
	users := NewService[user](c, "main", "users")
	ctx := context.Background()

	list, err := users.FindMany(ctx, Params{Where: W().Eq("active", true), Take: Take(10)})
	if err != nil || len(list) != 2 || list[1].Name != "b" {
		t.Fatalf("%v %v", list, err)
	}
	if got.Database != "main" || got.Service != "users" || got.Instance != "eu" || *got.Params.Take != 10 {
		t.Fatalf("query sent %+v", got)
	}
	n, err := users.Count(ctx, Where{})
	if err != nil || n != 2 {
		t.Fatalf("count %d %v", n, err)
	}
	_, err = users.FindOne(ctx, W().Eq("id", 9))
	var re *RemoteError
	if !errors.Is(err, ErrNotFound) || !errors.As(err, &re) || re.RequestID != "r1" || re.Code != "E404-1" {
		t.Fatalf("err %v", err)
	}
}

func TestMethods(t *testing.T) {
	if !FindMany.Valid() || Method("drop").Valid() || FindMany.Writes() || !DeleteMany.Writes() {
		t.Fatal("method classification wrong")
	}
	if (Query{Database: "main", Instance: "eu", Service: "users", Method: Count}).String() != "main@eu.users.count" {
		t.Fatal("query string")
	}
}
