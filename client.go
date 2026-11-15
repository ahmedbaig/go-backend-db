package db

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	PathQuery        = "/api/v1/query"
	PathTransaction  = "/api/v1/transaction"
	PathCapabilities = "/api/v1/capabilities"
)

type RemoteError struct {
	Status        int
	Code          string `json:"code"`
	Message       string `json:"message"`
	RequestID     string
	CorrelationID string
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("db-handler %d %s: %s", e.Status, e.Code, e.Message)
}

var (
	ErrNotFound   = errors.New("record not found")
	ErrConflict   = errors.New("record conflicts with an existing one")
	ErrValidation = errors.New("query rejected")
)

func (e *RemoteError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.Status == http.StatusNotFound
	case ErrConflict:
		return e.Status == http.StatusConflict
	case ErrValidation:
		return e.Status == http.StatusBadRequest
	}
	return false
}

type Client struct {
	base     string
	http     *http.Client
	instance string
	headers  http.Header
	t        Transport
}

type ClientOption func(*Client)

func WithHTTPClient(h *http.Client) ClientOption { return func(c *Client) { c.http = h } }

func WithInstance(name string) ClientOption { return func(c *Client) { c.instance = name } }

func WithHeader(k, v string) ClientOption { return func(c *Client) { c.headers.Set(k, v) } }

func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{base: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 30 * time.Second}, headers: http.Header{}}
	for _, o := range opts {
		o(c)
	}
	if c.t == nil {
		c.t = httpTransport{c}
	}
	return c
}

func NewClientWith(t Transport, opts ...ClientOption) *Client {
	return NewClient("", append(opts, func(c *Client) { c.t = t })...)
}

func (c *Client) Query(ctx context.Context, q Query) (Result, error) {
	if q.Instance == "" {
		q.Instance = c.instance
	}
	return c.t.Query(ctx, q)
}

func (c *Client) Transaction(ctx context.Context, tx Transaction) (TransactionResult, error) {
	if tx.Instance == "" {
		tx.Instance = c.instance
	}
	return c.t.Transaction(ctx, tx)
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	return c.t.Capabilities(ctx)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	for k, vs := range c.headers {
		req.Header[k] = vs
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		var env struct {
			RequestID     string        `json:"requestId"`
			CorrelationID string        `json:"correlationId"`
			Errors        []RemoteError `json:"errors"`
		}
		re := &RemoteError{Status: res.StatusCode, Message: strings.TrimSpace(string(body))}
		if json.Unmarshal(body, &env) == nil && len(env.Errors) > 0 {
			re = &env.Errors[0]
			re.Status = res.StatusCode
			re.RequestID, re.CorrelationID = env.RequestID, env.CorrelationID
		}
		return re
	}
	return json.Unmarshal(body, out)
}

type Service[T any] struct {
	c        *Client
	database string
	service  string
}

func NewService[T any](c *Client, database, service string) *Service[T] {
	return &Service[T]{c: c, database: database, service: service}
}

func (s *Service[T]) query(ctx context.Context, m Method, p Params) (Result, error) {
	return s.c.Query(ctx, Query{Database: s.database, Service: s.service, Method: m, Params: p})
}

func decode[V any](r Result) (V, error) {
	var v V
	if len(r.Data) == 0 || string(r.Data) == "null" {
		return v, nil
	}
	err := json.Unmarshal(r.Data, &v)
	return v, err
}

func count(r Result) int64 {
	if r.Count == nil {
		return 0
	}
	return *r.Count
}

func (s *Service[T]) FindOne(ctx context.Context, where Where, opts ...Params) (T, error) {
	r, err := s.query(ctx, FindOne, merge(Params{Where: where}, opts))
	if err != nil {
		var zero T
		return zero, err
	}
	return decode[T](r)
}

func (s *Service[T]) FindFirst(ctx context.Context, p Params) (T, error) {
	r, err := s.query(ctx, FindFirst, p)
	if err != nil {
		var zero T
		return zero, err
	}
	return decode[T](r)
}

func (s *Service[T]) FindMany(ctx context.Context, p Params) ([]T, error) {
	r, err := s.query(ctx, FindMany, p)
	if err != nil {
		return nil, err
	}
	return decode[[]T](r)
}

func (s *Service[T]) Count(ctx context.Context, where Where) (int64, error) {
	r, err := s.query(ctx, Count, Params{Where: where})
	return count(r), err
}

func (s *Service[T]) Create(ctx context.Context, v T) (T, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return v, err
	}
	r, err := s.query(ctx, Create, Params{Data: data})
	if err != nil {
		return v, err
	}
	return decode[T](r)
}

func (s *Service[T]) CreateMany(ctx context.Context, vs []T) (int64, error) {
	data, err := json.Marshal(vs)
	if err != nil {
		return 0, err
	}
	r, err := s.query(ctx, CreateMany, Params{Data: data})
	return count(r), err
}

func (s *Service[T]) Update(ctx context.Context, where Where, changes any) (T, error) {
	data, err := json.Marshal(changes)
	if err != nil {
		var zero T
		return zero, err
	}
	r, err := s.query(ctx, Update, Params{Where: where, Data: data})
	if err != nil {
		var zero T
		return zero, err
	}
	return decode[T](r)
}

func (s *Service[T]) UpdateMany(ctx context.Context, where Where, changes any) (int64, error) {
	data, err := json.Marshal(changes)
	if err != nil {
		return 0, err
	}
	r, err := s.query(ctx, UpdateMany, Params{Where: where, Data: data})
	return count(r), err
}

func (s *Service[T]) Upsert(ctx context.Context, conflict []string, create T, update any) (T, error) {
	c, err := json.Marshal(create)
	if err != nil {
		return create, err
	}
	u, err := json.Marshal(update)
	if err != nil {
		return create, err
	}
	r, err := s.query(ctx, Upsert, Params{Conflict: conflict, Create: c, Data: u})
	if err != nil {
		return create, err
	}
	return decode[T](r)
}

func (s *Service[T]) Delete(ctx context.Context, where Where) (T, error) {
	r, err := s.query(ctx, Delete, Params{Where: where})
	if err != nil {
		var zero T
		return zero, err
	}
	return decode[T](r)
}

func (s *Service[T]) DeleteMany(ctx context.Context, where Where) (int64, error) {
	r, err := s.query(ctx, DeleteMany, Params{Where: where})
	return count(r), err
}

func merge(p Params, extra []Params) Params {
	for _, e := range extra {
		if len(e.Select) > 0 {
			p.Select = e.Select
		}
		if len(e.Include) > 0 {
			p.Include = e.Include
		}
	}
	return p
}
