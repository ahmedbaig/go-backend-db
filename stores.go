package db

import (
	"encoding/json"
	"time"
)

type KVOp string

const (
	KVGet     KVOp = "get"
	KVMGet    KVOp = "mget"
	KVSet     KVOp = "set"
	KVDel     KVOp = "del"
	KVExpire  KVOp = "expire"
	KVTTL     KVOp = "ttl"
	KVIncr    KVOp = "incr"
	KVPublish KVOp = "publish"
)

var KVOps = []KVOp{KVGet, KVMGet, KVSet, KVDel, KVExpire, KVTTL, KVIncr, KVPublish}

type KVRequest struct {
	Key     string          `json:"key,omitempty"`
	Keys    []string        `json:"keys,omitempty"`
	Value   json.RawMessage `json:"value,omitempty"`
	TTLms   int64           `json:"ttlMs,omitempty"`
	NX      bool            `json:"nx,omitempty"`
	XX      bool            `json:"xx,omitempty"`
	Touch   bool            `json:"touch,omitempty"`
	By      int64           `json:"by,omitempty"`
	Channel string          `json:"channel,omitempty"`
}

type KVResult struct {
	Found  bool              `json:"found"`
	Value  json.RawMessage   `json:"value,omitempty"`
	Values []json.RawMessage `json:"values,omitempty"`
	TTLms  int64             `json:"ttlMs,omitempty"`
	Count  int64             `json:"count"`
	OK     bool              `json:"ok"`
}

type DocsOp string

const (
	DocsInsert    DocsOp = "insert"
	DocsFind      DocsOp = "find"
	DocsFindOne   DocsOp = "findOne"
	DocsCount     DocsOp = "count"
	DocsAggregate DocsOp = "aggregate"
	DocsUpdate    DocsOp = "update"
	DocsDelete    DocsOp = "delete"
)

var DocsOps = []DocsOp{DocsInsert, DocsFind, DocsFindOne, DocsCount, DocsAggregate, DocsUpdate, DocsDelete}

type DocsRequest struct {
	Filter     json.RawMessage   `json:"filter,omitempty"`
	Sort       json.RawMessage   `json:"sort,omitempty"`
	Projection json.RawMessage   `json:"projection,omitempty"`
	Limit      int64             `json:"limit,omitempty"`
	Skip       int64             `json:"skip,omitempty"`
	Documents  []json.RawMessage `json:"documents,omitempty"`
	Update     json.RawMessage   `json:"update,omitempty"`
	Upsert     bool              `json:"upsert,omitempty"`
	Many       bool              `json:"many,omitempty"`
	Pipeline   []json.RawMessage `json:"pipeline,omitempty"`
}

type DocsResult struct {
	Data        json.RawMessage   `json:"data,omitempty"`
	Count       int64             `json:"count"`
	InsertedIDs []json.RawMessage `json:"insertedIds,omitempty"`
	Matched     int64             `json:"matched,omitempty"`
	Modified    int64             `json:"modified,omitempty"`
	UpsertedID  json.RawMessage   `json:"upsertedId,omitempty"`
}

type Event struct {
	Name      string         `json:"name" validate:"required,max=128"`
	UserID    string         `json:"userId,omitempty" validate:"max=128"`
	SessionID string         `json:"sessionId,omitempty" validate:"max=128"`
	Props     map[string]any `json:"props,omitempty"`
	At        time.Time      `json:"at,omitzero"`
}

type TrackRequest struct {
	Events []Event `json:"events" validate:"required,min=1,max=500,dive"`
}

type TrackResult struct {
	Accepted int `json:"accepted"`
}

type DAUPoint struct {
	Day   string `json:"day"`
	Users int64  `json:"users"`
}

type DAURequest struct {
	From string `query:"from" validate:"omitempty,datetime=2006-01-02"`
	To   string `query:"to" validate:"omitempty,datetime=2006-01-02"`
}

type StoreInfo struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Prefix string `json:"prefix,omitempty"`
	DB     string `json:"database,omitempty"`
}
