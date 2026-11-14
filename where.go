package db

import (
	"encoding/json"
	"fmt"
)

type Operator string

const (
	OpEq         Operator = "eq"
	OpNe         Operator = "ne"
	OpGt         Operator = "gt"
	OpGte        Operator = "gte"
	OpLt         Operator = "lt"
	OpLte        Operator = "lte"
	OpIn         Operator = "in"
	OpNotIn      Operator = "notIn"
	OpContains   Operator = "contains"
	OpStartsWith Operator = "startsWith"
	OpEndsWith   Operator = "endsWith"
	OpIsNull     Operator = "isNull"
)

var Operators = []Operator{OpEq, OpNe, OpGt, OpGte, OpLt, OpLte, OpIn, OpNotIn, OpContains, OpStartsWith, OpEndsWith, OpIsNull}

func (o Operator) Valid() bool {
	for _, v := range Operators {
		if v == o {
			return true
		}
	}
	return false
}

type Condition map[Operator]any

type Where struct {
	Fields map[string]Condition `json:"-"`
	And    []Where              `json:"AND,omitempty"`
	Or     []Where              `json:"OR,omitempty"`
	Not    *Where               `json:"NOT,omitempty"`
}

func (w Where) IsZero() bool {
	return len(w.Fields) == 0 && len(w.And) == 0 && len(w.Or) == 0 && w.Not == nil
}

func (w Where) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	for k, c := range w.Fields {
		m[k] = c
	}
	if len(w.And) > 0 {
		m["AND"] = w.And
	}
	if len(w.Or) > 0 {
		m["OR"] = w.Or
	}
	if w.Not != nil {
		m["NOT"] = w.Not
	}
	return json.Marshal(m)
}

func (w *Where) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*w = Where{}
	for k, v := range raw {
		switch k {
		case "AND":
			if err := json.Unmarshal(v, &w.And); err != nil {
				return fmt.Errorf("AND: %w", err)
			}
		case "OR":
			if err := json.Unmarshal(v, &w.Or); err != nil {
				return fmt.Errorf("OR: %w", err)
			}
		case "NOT":
			w.Not = &Where{}
			if err := json.Unmarshal(v, w.Not); err != nil {
				return fmt.Errorf("NOT: %w", err)
			}
		default:
			c, err := parseCondition(v)
			if err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
			if w.Fields == nil {
				w.Fields = map[string]Condition{}
			}
			w.Fields[k] = c
		}
	}
	return nil
}

func parseCondition(v json.RawMessage) (Condition, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(v, &obj); err == nil && len(obj) > 0 && allOperators(obj) {
		c := Condition{}
		for k, raw := range obj {
			var val any
			if err := json.Unmarshal(raw, &val); err != nil {
				return nil, err
			}
			c[Operator(k)] = val
		}
		return c, nil
	}
	var val any
	if err := json.Unmarshal(v, &val); err != nil {
		return nil, err
	}
	if val == nil {
		return Condition{OpIsNull: true}, nil
	}
	if _, isObj := val.(map[string]any); isObj {
		return nil, fmt.Errorf("unknown operator in %s", string(v))
	}
	return Condition{OpEq: val}, nil
}

func allOperators(m map[string]json.RawMessage) bool {
	for k := range m {
		if !Operator(k).Valid() {
			return false
		}
	}
	return true
}

func W() Where { return Where{Fields: map[string]Condition{}} }

func (w Where) Field(name string, c Condition) Where {
	if w.Fields == nil {
		w.Fields = map[string]Condition{}
	}
	w.Fields[name] = c
	return w
}

func (w Where) Eq(field string, v any) Where { return w.Field(field, Condition{OpEq: v}) }

func AndOf(ws ...Where) Where { return Where{And: ws} }

func OrOf(ws ...Where) Where { return Where{Or: ws} }

func NotOf(w Where) Where { return Where{Not: &w} }

func Eq(v any) Condition            { return Condition{OpEq: v} }
func Ne(v any) Condition            { return Condition{OpNe: v} }
func Gt(v any) Condition            { return Condition{OpGt: v} }
func Gte(v any) Condition           { return Condition{OpGte: v} }
func Lt(v any) Condition            { return Condition{OpLt: v} }
func Lte(v any) Condition           { return Condition{OpLte: v} }
func In(v ...any) Condition         { return Condition{OpIn: v} }
func NotIn(v ...any) Condition      { return Condition{OpNotIn: v} }
func Contains(s string) Condition   { return Condition{OpContains: s} }
func StartsWith(s string) Condition { return Condition{OpStartsWith: s} }
func EndsWith(s string) Condition   { return Condition{OpEndsWith: s} }
func IsNull(b bool) Condition       { return Condition{OpIsNull: b} }

func Between(lo, hi any) Condition { return Condition{OpGte: lo, OpLte: hi} }

func Take(n int) *int { return &n }
