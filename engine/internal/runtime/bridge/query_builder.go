package bridge

import (
	"fmt"
)

type queryBuilder struct {
	handle     ModelHandle
	conditions [][]any
	order      string
	limit      int
	offset     int
}

func buildFluentModelProxy(handle ModelHandle) map[string]any {
	return map[string]any{
		"where": func(params ...any) any {
			qb := &queryBuilder{handle: handle}
			qb.applyWhere(params)
			return buildQueryProxyMap(qb)
		},
		"find": func(id any) any {
			return buildRecordProxyMap(handle, id)
		},
		"orderBy": func(params ...any) any {
			qb := &queryBuilder{handle: handle}
			if len(params) > 0 {
				if field, ok := params[0].(string); ok {
					qb.order = field
					if len(params) > 1 {
						if dir, ok := params[1].(string); ok {
							qb.order = field + " " + dir
						}
					}
				}
			}
			return buildQueryProxyMap(qb)
		},
		"limit": func(n any) any {
			qb := &queryBuilder{handle: handle}
			qb.limit = toIntSafe(n)
			return buildQueryProxyMap(qb)
		},
	}
}

func buildQueryProxyMap(qb *queryBuilder) map[string]any {
	return map[string]any{
		"where": func(params ...any) any {
			qb.applyWhere(params)
			return buildQueryProxyMap(qb)
		},
		"orderBy": func(params ...any) any {
			if len(params) > 0 {
				if field, ok := params[0].(string); ok {
					qb.order = field
					if len(params) > 1 {
						if dir, ok := params[1].(string); ok {
							qb.order = field + " " + dir
						}
					}
				}
			}
			return buildQueryProxyMap(qb)
		},
		"limit": func(n any) any {
			qb.limit = toIntSafe(n)
			return buildQueryProxyMap(qb)
		},
		"offset": func(n any) any {
			qb.offset = toIntSafe(n)
			return buildQueryProxyMap(qb)
		},
		"get": func() (any, error) {
			return qb.executeGet()
		},
		"first": func() (any, error) {
			qb.limit = 1
			return qb.executeFirst()
		},
		"count": func() (any, error) {
			return qb.executeCount()
		},
		"sum": func(field ...any) (any, error) {
			if len(field) == 0 {
				return nil, fmt.Errorf("sum: field name required")
			}
			fieldName, _ := field[0].(string)
			if fieldName == "" {
				return nil, fmt.Errorf("sum: field name must be a non-empty string")
			}
			return qb.executeSum(fieldName)
		},
	}
}

func buildRecordProxyMap(handle ModelHandle, id any) map[string]any {
	idStr := fmt.Sprintf("%v", id)
	return map[string]any{
		"get": func() (any, error) {
			r, err := handle.Get(idStr)
			return convertToAny(r), err
		},
		"update": func(data map[string]any) (any, error) {
			return nil, handle.Write(idStr, data)
		},
		"delete": func() (any, error) {
			return nil, handle.Delete(idStr)
		},
	}
}

// applyWhere supports: where("f", val), where("f", "op", val)
func (qb *queryBuilder) applyWhere(params []any) {
	if len(params) == 0 {
		return
	}
	field, ok := params[0].(string)
	if !ok {
		return
	}

	switch len(params) {
	case 2:
		qb.conditions = append(qb.conditions, []any{field, "=", params[1]})
	case 3:
		op, _ := params[1].(string)
		if op == "" {
			op = "="
		}
		qb.conditions = append(qb.conditions, []any{field, op, params[2]})
	}
}

func (qb *queryBuilder) executeGet() (any, error) {
	opts := qb.toSearchOptions()
	results, err := qb.handle.Search(opts)
	if err != nil {
		return nil, err
	}
	if results == nil {
		return []any{}, nil
	}
	return convertToAny(results), nil
}

func (qb *queryBuilder) executeFirst() (any, error) {
	opts := qb.toSearchOptions()
	opts.Limit = 1
	results, err := qb.handle.Search(opts)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return convertToAny(results[0]), nil
}

func (qb *queryBuilder) executeCount() (any, error) {
	opts := qb.toSearchOptions()
	count, err := qb.handle.Count(opts)
	if err != nil {
		return nil, err
	}
	return count, nil
}

func (qb *queryBuilder) executeSum(field string) (any, error) {
	opts := qb.toSearchOptions()
	sum, err := qb.handle.Sum(field, opts)
	if err != nil {
		return nil, err
	}
	return sum, nil
}

func (qb *queryBuilder) toSearchOptions() SearchOptions {
	opts := SearchOptions{}
	if len(qb.conditions) > 0 {
		opts.Domain = qb.conditions
	}
	if qb.order != "" {
		opts.Order = qb.order
	}
	if qb.limit > 0 {
		opts.Limit = qb.limit
	}
	if qb.offset > 0 {
		opts.Offset = qb.offset
	}
	return opts
}

func toIntSafe(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	case int32:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
