package dbgrpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/ahmedbaig/go-backend-db"
	"github.com/ahmedbaig/go-backend-db/dbgrpc"
	"github.com/ahmedbaig/go-backend-http"
	dbv1 "github.com/ahmedbaig/go-backend-proto/gen/go/ahmedbaig/dbhandler/v1"
	"github.com/ahmedbaig/go-backend-proto/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type post struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type fake struct {
	dbv1.UnimplementedDBHandlerServiceServer
	last *dbv1.QueryRequest
}

func (f *fake) Query(_ context.Context, q *dbv1.QueryRequest) (*dbv1.QueryResponse, error) {
	f.last = q
	var p db.Params
	if err := json.Unmarshal(q.GetParamsJson(), &p); err != nil {
		return nil, err
	}
	if id, _ := p.Where.Fields["id"][db.OpEq].(float64); id == 404 {
		return nil, httpx.E(httpx.CodeNotFound)
	}
	n := int64(1)
	switch db.Method(q.GetMethod()) {
	case db.Count:
		return &dbv1.QueryResponse{Count: &n}, nil
	case db.Create:
		return &dbv1.QueryResponse{DataJson: p.Data}, nil
	}
	return &dbv1.QueryResponse{DataJson: []byte(`{"id":7,"title":"hi"}`)}, nil
}

func (f *fake) KeyValue(_ context.Context, r *dbv1.KeyValueRequest) (*dbv1.KeyValueResponse, error) {
	var req db.KVRequest
	_ = json.Unmarshal(r.GetRequestJson(), &req)
	out, _ := json.Marshal(db.KVResult{Found: true, Value: json.RawMessage(`"` + r.GetStore() + ":" + req.Key + `"`)})
	return &dbv1.KeyValueResponse{ResultJson: out}, nil
}

func client(t *testing.T) (*db.Client, *fake) {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	f := &fake{}
	srv := grpcx.NewServer(grpcx.ServerOptions{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	dbv1.RegisterDBHandlerServiceServer(srv, f)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	c, conn, err := dbgrpc.Dial("passthrough:///bufnet", grpcx.ClientOptions{Extra: []grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
	}}, db.WithInstance("primary"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return c, f
}

func TestServiceOverGRPC(t *testing.T) {
	c, f := client(t)
	posts := db.NewService[post](c, "demo", "post")
	ctx := context.Background()

	p, err := posts.FindOne(ctx, db.W().Eq("id", 7))
	if err != nil || p.Title != "hi" {
		t.Fatalf("%+v %v", p, err)
	}
	if f.last.GetInstance() != "primary" || f.last.GetService() != "post" || f.last.GetMethod() != "findOne" {
		t.Fatalf("routing %+v", f.last)
	}
	n, err := posts.Count(ctx, db.Where{})
	if err != nil || n != 1 {
		t.Fatalf("count %d %v", n, err)
	}
	created, err := posts.Create(ctx, post{ID: 9, Title: "new"})
	if err != nil || created.ID != 9 {
		t.Fatalf("create %+v %v", created, err)
	}
}

func TestErrorsKeepMeaning(t *testing.T) {
	c, _ := client(t)
	_, err := db.NewService[post](c, "demo", "post").FindOne(context.Background(), db.W().Eq("id", 404))
	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	var re *db.RemoteError
	if !errors.As(err, &re) || re.Code != string(httpx.CodeNotFound) {
		t.Fatalf("got %#v", err)
	}
}

func TestKVOverGRPC(t *testing.T) {
	c, _ := client(t)
	v, found, err := c.KV("sessions").Get(context.Background(), "abc")
	if err != nil || !found || string(v) != `"sessions:abc"` {
		t.Fatalf("%s %v %v", v, found, err)
	}
}
