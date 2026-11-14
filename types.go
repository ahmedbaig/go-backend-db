package db

import (
	"encoding/json"
	"fmt"
)

type Method string

const (
	FindOne    Method = "findOne"
	FindFirst  Method = "findFirst"
	FindMany   Method = "findMany"
	Count      Method = "count"
	Create     Method = "create"
	CreateMany Method = "createMany"
	Update     Method = "update"
	UpdateMany Method = "updateMany"
	Upsert     Method = "upsert"
	Delete     Method = "delete"
	DeleteMany Method = "deleteMany"
)

var Methods = []Method{FindOne, FindFirst, FindMany, Count, Create, CreateMany, Update, UpdateMany, Upsert, Delete, DeleteMany}

func (m Method) Valid() bool {
	for _, v := range Methods {
		if v == m {
			return true
		}
	}
	return false
}

func (m Method) Writes() bool {
	switch m {
	case Create, CreateMany, Update, UpdateMany, Upsert, Delete, DeleteMany:
		return true
	}
	return false
}

type Direction string

const (
	Asc  Direction = "asc"
	Desc Direction = "desc"
)

type Order struct {
	Field     string    `json:"field"`
	Direction Direction `json:"direction,omitempty"`
}

type Params struct {
	Where    Where           `json:"where,omitzero"`
	Select   []string        `json:"select,omitempty"`
	Include  []string        `json:"include,omitempty"`
	OrderBy  []Order         `json:"orderBy,omitempty"`
	Take     *int            `json:"take,omitempty"`
	Skip     int             `json:"skip,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	Create   json.RawMessage `json:"create,omitempty"`
	Conflict []string        `json:"conflict,omitempty"`
}

type Query struct {
	Database string `json:"database" validate:"required"`
	Instance string `json:"instance,omitempty"`
	Service  string `json:"service" validate:"required"`
	Method   Method `json:"method" validate:"required"`
	Params   Params `json:"params"`
}

func (q Query) String() string {
	s := q.Database
	if q.Instance != "" {
		s += "@" + q.Instance
	}
	return fmt.Sprintf("%s.%s.%s", s, q.Service, q.Method)
}

type Transaction struct {
	Database string  `json:"database" validate:"required"`
	Instance string  `json:"instance,omitempty"`
	Queries  []Query `json:"queries" validate:"required,min=1,max=100"`
}

type Result struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Count *int64          `json:"count,omitempty"`
}

type TransactionResult struct {
	Results []Result `json:"results"`
}

type Field struct {
	Name       string `json:"name"`
	Column     string `json:"column"`
	Type       string `json:"type"`
	PrimaryKey bool   `json:"primaryKey,omitempty"`
	Nullable   bool   `json:"nullable,omitempty"`
	Unique     bool   `json:"unique,omitempty"`
}

type ServiceInfo struct {
	Name      string   `json:"name"`
	Table     string   `json:"table"`
	Fields    []Field  `json:"fields"`
	Relations []string `json:"relations,omitempty"`
	ReadOnly  bool     `json:"readOnly,omitempty"`
}

type DatabaseInfo struct {
	Name      string        `json:"name"`
	Driver    string        `json:"driver"`
	Instances []string      `json:"instances,omitempty"`
	Services  []ServiceInfo `json:"services"`
}

type Capabilities struct {
	Methods   []Method       `json:"methods"`
	Operators []Operator     `json:"operators"`
	MaxTake   int            `json:"maxTake"`
	Databases []DatabaseInfo `json:"databases"`
	Stores    []StoreInfo    `json:"stores,omitempty"`
	KVOps     []KVOp         `json:"kvOps,omitempty"`
	DocsOps   []DocsOp       `json:"docsOps,omitempty"`
}
