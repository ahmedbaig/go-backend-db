package db

import (
	"context"
	"encoding/json"
	"net/url"
	"time"
)

type M map[string]any

func PathDocs(store, collection string, op DocsOp) string {
	return "/api/v1/docs/" + url.PathEscape(store) + "/" + url.PathEscape(collection) + "/" + string(op)
}

func PathTrack(store string) string { return "/api/v1/analytics/" + url.PathEscape(store) + "/track" }

func PathDAU(store string) string { return "/api/v1/analytics/" + url.PathEscape(store) + "/dau" }

type Docs struct {
	c          *Client
	store      string
	collection string
}

func (c *Client) Docs(store, collection string) *Docs {
	return &Docs{c: c, store: store, collection: collection}
}

type FindOptions struct {
	Sort       M
	Projection M
	Limit      int64
	Skip       int64
}

func raw(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	if r, ok := v.(json.RawMessage); ok {
		return r
	}
	b, _ := json.Marshal(v)
	if string(b) == "null" {
		return nil
	}
	return b
}

func (d *Docs) do(ctx context.Context, op DocsOp, req DocsRequest) (DocsResult, error) {
	return d.c.t.Docs(ctx, d.store, d.collection, op, req)
}

func (d *Docs) Insert(ctx context.Context, docs ...any) ([]json.RawMessage, error) {
	req := DocsRequest{Documents: make([]json.RawMessage, len(docs))}
	for i, doc := range docs {
		b, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}
		req.Documents[i] = b
	}
	r, err := d.do(ctx, DocsInsert, req)
	return r.InsertedIDs, err
}

func (d *Docs) Count(ctx context.Context, filter M) (int64, error) {
	r, err := d.do(ctx, DocsCount, DocsRequest{Filter: raw(filter)})
	return r.Count, err
}

func (d *Docs) Update(ctx context.Context, filter, update M, many, upsert bool) (DocsResult, error) {
	return d.do(ctx, DocsUpdate, DocsRequest{Filter: raw(filter), Update: raw(update), Many: many, Upsert: upsert})
}

func (d *Docs) Delete(ctx context.Context, filter M, many bool) (int64, error) {
	r, err := d.do(ctx, DocsDelete, DocsRequest{Filter: raw(filter), Many: many})
	return r.Count, err
}

func FindDocs[T any](ctx context.Context, d *Docs, filter M, o FindOptions) ([]T, error) {
	r, err := d.do(ctx, DocsFind, DocsRequest{Filter: raw(filter), Sort: raw(o.Sort), Projection: raw(o.Projection), Limit: o.Limit, Skip: o.Skip})
	if err != nil {
		return nil, err
	}
	return decode[[]T](Result{Data: r.Data})
}

func FindOneDoc[T any](ctx context.Context, d *Docs, filter M) (T, error) {
	r, err := d.do(ctx, DocsFindOne, DocsRequest{Filter: raw(filter)})
	if err != nil {
		var zero T
		return zero, err
	}
	return decode[T](Result{Data: r.Data})
}

func Aggregate[T any](ctx context.Context, d *Docs, pipeline ...M) ([]T, error) {
	req := DocsRequest{Pipeline: make([]json.RawMessage, len(pipeline))}
	for i, st := range pipeline {
		req.Pipeline[i] = raw(st)
	}
	r, err := d.do(ctx, DocsAggregate, req)
	if err != nil {
		return nil, err
	}
	return decode[[]T](Result{Data: r.Data})
}

type Analytics struct {
	c     *Client
	store string
}

func (c *Client) Analytics(store string) *Analytics { return &Analytics{c: c, store: store} }

func (a *Analytics) Track(ctx context.Context, events ...Event) (int, error) {
	r, err := a.c.t.Track(ctx, a.store, TrackRequest{Events: events})
	return r.Accepted, err
}

func (a *Analytics) DAU(ctx context.Context, from, to time.Time) ([]DAUPoint, error) {
	var f, t string
	if !from.IsZero() {
		f = from.Format(time.DateOnly)
	}
	if !to.IsZero() {
		t = to.Format(time.DateOnly)
	}
	return a.c.t.DAU(ctx, a.store, f, t)
}
