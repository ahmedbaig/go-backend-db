package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"time"
)

func PathKV(store string, op KVOp) string {
	return "/api/v1/kv/" + url.PathEscape(store) + "/" + string(op)
}

type KV struct {
	c     *Client
	store string
}

func (c *Client) KV(store string) *KV { return &KV{c: c, store: store} }

func (k *KV) do(ctx context.Context, op KVOp, req KVRequest) (KVResult, error) {
	var r KVResult
	err := k.c.post(ctx, PathKV(k.store, op), req, &r)
	return r, err
}

func (k *KV) Get(ctx context.Context, key string) (json.RawMessage, bool, error) {
	r, err := k.do(ctx, KVGet, KVRequest{Key: key})
	return r.Value, r.Found, err
}

func (k *KV) GetTouch(ctx context.Context, key string, ttl time.Duration) (json.RawMessage, bool, error) {
	r, err := k.do(ctx, KVGet, KVRequest{Key: key, Touch: true, TTLms: ttl.Milliseconds()})
	return r.Value, r.Found, err
}

func (k *KV) MGet(ctx context.Context, keys ...string) ([]json.RawMessage, error) {
	r, err := k.do(ctx, KVMGet, KVRequest{Keys: keys})
	return r.Values, err
}

func (k *KV) Set(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = k.do(ctx, KVSet, KVRequest{Key: key, Value: b, TTLms: ttl.Milliseconds()})
	return err
}

func (k *KV) SetNX(ctx context.Context, key string, v any, ttl time.Duration) (bool, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return false, err
	}
	r, err := k.do(ctx, KVSet, KVRequest{Key: key, Value: b, TTLms: ttl.Milliseconds(), NX: true})
	return r.OK, err
}

func (k *KV) Del(ctx context.Context, keys ...string) (int64, error) {
	r, err := k.do(ctx, KVDel, KVRequest{Keys: keys})
	return r.Count, err
}

func (k *KV) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	r, err := k.do(ctx, KVExpire, KVRequest{Key: key, TTLms: ttl.Milliseconds()})
	return r.OK, err
}

func (k *KV) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	r, err := k.do(ctx, KVTTL, KVRequest{Key: key})
	return time.Duration(r.TTLms) * time.Millisecond, r.Found, err
}

func (k *KV) Incr(ctx context.Context, key string, by int64, ttl time.Duration) (int64, error) {
	r, err := k.do(ctx, KVIncr, KVRequest{Key: key, By: by, TTLms: ttl.Milliseconds()})
	return r.Count, err
}

func (k *KV) Publish(ctx context.Context, channel string, msg any) (int64, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	r, err := k.do(ctx, KVPublish, KVRequest{Channel: channel, Value: b})
	return r.Count, err
}

func GetJSON[T any](ctx context.Context, k *KV, key string) (T, bool, error) {
	var v T
	raw, found, err := k.Get(ctx, key)
	if err != nil || !found {
		return v, found, err
	}
	return v, true, json.Unmarshal(raw, &v)
}

type Sessions[T any] struct {
	kv     *KV
	prefix string
	ttl    time.Duration
}

func NewSessions[T any](kv *KV, prefix string, ttl time.Duration) *Sessions[T] {
	return &Sessions[T]{kv: kv, prefix: prefix, ttl: ttl}
}

func (s *Sessions[T]) key(id string) string { return s.prefix + id }

func NewSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Sessions[T]) Create(ctx context.Context, data T) (string, error) {
	for range 3 {
		id, err := NewSessionID()
		if err != nil {
			return "", err
		}
		ok, err := s.kv.SetNX(ctx, s.key(id), data, s.ttl)
		if err != nil {
			return "", err
		}
		if ok {
			return id, nil
		}
	}
	return "", ErrConflict
}

func (s *Sessions[T]) Get(ctx context.Context, id string) (T, bool, error) {
	var v T
	raw, found, err := s.kv.GetTouch(ctx, s.key(id), s.ttl)
	if err != nil || !found {
		return v, found, err
	}
	return v, true, json.Unmarshal(raw, &v)
}

func (s *Sessions[T]) Save(ctx context.Context, id string, data T) error {
	return s.kv.Set(ctx, s.key(id), data, s.ttl)
}

func (s *Sessions[T]) Destroy(ctx context.Context, id string) error {
	_, err := s.kv.Del(ctx, s.key(id))
	return err
}
