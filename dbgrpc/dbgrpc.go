package dbgrpc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ahmedbaig/go-backend-db"
	"github.com/ahmedbaig/go-backend-http"
	dbv1 "github.com/ahmedbaig/go-backend-proto/gen/go/ahmedbaig/dbhandler/v1"
	"github.com/ahmedbaig/go-backend-proto/grpcx"
	"google.golang.org/grpc"
)

type Transport struct {
	c dbv1.DBHandlerServiceClient
}

func New(conn grpc.ClientConnInterface) *Transport {
	return &Transport{c: dbv1.NewDBHandlerServiceClient(conn)}
}

func Dial(target string, o grpcx.ClientOptions, opts ...db.ClientOption) (*db.Client, *grpc.ClientConn, error) {
	conn, err := grpcx.Dial(target, o)
	if err != nil {
		return nil, nil, err
	}
	return db.NewClientWith(New(conn), opts...), conn, nil
}

func remote(ctx context.Context, err error) error {
	e := grpcx.FromStatus(err)
	status := e.Status
	if status == 0 {
		status = http.StatusBadGateway
	}
	return &db.RemoteError{
		Status:        status,
		Code:          string(e.Code),
		Message:       e.Message,
		RequestID:     httpx.RequestID(ctx),
		CorrelationID: httpx.CorrelationID(ctx),
	}
}

func queryRequest(q db.Query) (*dbv1.QueryRequest, error) {
	params, err := json.Marshal(q.Params)
	if err != nil {
		return nil, err
	}
	return &dbv1.QueryRequest{Database: q.Database, Instance: q.Instance, Service: q.Service, Method: string(q.Method), ParamsJson: params}, nil
}

func (t *Transport) Query(ctx context.Context, q db.Query) (db.Result, error) {
	req, err := queryRequest(q)
	if err != nil {
		return db.Result{}, err
	}
	res, err := t.c.Query(ctx, req)
	if err != nil {
		return db.Result{}, remote(ctx, err)
	}
	return db.Result{Data: res.GetDataJson(), Count: res.Count}, nil
}

func (t *Transport) Transaction(ctx context.Context, tx db.Transaction) (db.TransactionResult, error) {
	req := &dbv1.TransactionRequest{Database: tx.Database, Instance: tx.Instance}
	for _, q := range tx.Queries {
		r, err := queryRequest(q)
		if err != nil {
			return db.TransactionResult{}, err
		}
		req.Queries = append(req.Queries, r)
	}
	res, err := t.c.Transaction(ctx, req)
	if err != nil {
		return db.TransactionResult{}, remote(ctx, err)
	}
	out := db.TransactionResult{Results: make([]db.Result, 0, len(res.GetResults()))}
	for _, r := range res.GetResults() {
		out.Results = append(out.Results, db.Result{Data: r.GetDataJson(), Count: r.Count})
	}
	return out, nil
}

func (t *Transport) Capabilities(ctx context.Context) (db.Capabilities, error) {
	var out db.Capabilities
	res, err := t.c.Capabilities(ctx, &dbv1.CapabilitiesRequest{})
	if err != nil {
		return out, remote(ctx, err)
	}
	return out, json.Unmarshal(res.GetCapabilitiesJson(), &out)
}

func (t *Transport) KV(ctx context.Context, store string, op db.KVOp, req db.KVRequest) (db.KVResult, error) {
	var out db.KVResult
	b, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	res, err := t.c.KeyValue(ctx, &dbv1.KeyValueRequest{Store: store, Op: string(op), RequestJson: b})
	if err != nil {
		return out, remote(ctx, err)
	}
	return out, json.Unmarshal(res.GetResultJson(), &out)
}

func (t *Transport) Docs(ctx context.Context, store, collection string, op db.DocsOp, req db.DocsRequest) (db.DocsResult, error) {
	var out db.DocsResult
	b, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	res, err := t.c.Documents(ctx, &dbv1.DocumentsRequest{Store: store, Collection: collection, Op: string(op), RequestJson: b})
	if err != nil {
		return out, remote(ctx, err)
	}
	return out, json.Unmarshal(res.GetResultJson(), &out)
}

func (t *Transport) Track(ctx context.Context, store string, req db.TrackRequest) (db.TrackResult, error) {
	var out db.TrackResult
	b, err := json.Marshal(req.Events)
	if err != nil {
		return out, err
	}
	res, err := t.c.Track(ctx, &dbv1.TrackRequest{Store: store, EventsJson: b})
	if err != nil {
		return out, remote(ctx, err)
	}
	return out, json.Unmarshal(res.GetResultJson(), &out)
}

func (t *Transport) DAU(ctx context.Context, store, from, to string) ([]db.DAUPoint, error) {
	res, err := t.c.DailyActiveUsers(ctx, &dbv1.DailyActiveUsersRequest{Store: store, From: from, To: to})
	if err != nil {
		return nil, remote(ctx, err)
	}
	var out []db.DAUPoint
	return out, json.Unmarshal(res.GetPointsJson(), &out)
}
