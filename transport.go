package db

import (
	"context"
	"net/http"
	"net/url"
)

type Transport interface {
	Query(ctx context.Context, q Query) (Result, error)
	Transaction(ctx context.Context, tx Transaction) (TransactionResult, error)
	Capabilities(ctx context.Context) (Capabilities, error)
	KV(ctx context.Context, store string, op KVOp, req KVRequest) (KVResult, error)
	Docs(ctx context.Context, store, collection string, op DocsOp, req DocsRequest) (DocsResult, error)
	Track(ctx context.Context, store string, req TrackRequest) (TrackResult, error)
	DAU(ctx context.Context, store, from, to string) ([]DAUPoint, error)
}

type httpTransport struct{ c *Client }

func (h httpTransport) Query(ctx context.Context, q Query) (Result, error) {
	var r Result
	return r, h.c.post(ctx, PathQuery, q, &r)
}

func (h httpTransport) Transaction(ctx context.Context, tx Transaction) (TransactionResult, error) {
	var r TransactionResult
	return r, h.c.post(ctx, PathTransaction, tx, &r)
}

func (h httpTransport) Capabilities(ctx context.Context) (Capabilities, error) {
	var out Capabilities
	return out, h.c.get(ctx, PathCapabilities, &out)
}

func (h httpTransport) KV(ctx context.Context, store string, op KVOp, req KVRequest) (KVResult, error) {
	var r KVResult
	return r, h.c.post(ctx, PathKV(store, op), req, &r)
}

func (h httpTransport) Docs(ctx context.Context, store, collection string, op DocsOp, req DocsRequest) (DocsResult, error) {
	var r DocsResult
	return r, h.c.post(ctx, PathDocs(store, collection, op), req, &r)
}

func (h httpTransport) Track(ctx context.Context, store string, req TrackRequest) (TrackResult, error) {
	var r TrackResult
	return r, h.c.post(ctx, PathTrack(store), req, &r)
}

func (h httpTransport) DAU(ctx context.Context, store, from, to string) ([]DAUPoint, error) {
	q := url.Values{}
	if from != "" {
		q.Set("from", from)
	}
	if to != "" {
		q.Set("to", to)
	}
	path := PathDAU(store)
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out []DAUPoint
	return out, h.c.get(ctx, path, &out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}
