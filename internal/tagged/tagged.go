// Package tagged converts evaluated TOML into toml-test's tagged JSON form,
// where every scalar is {"type": ..., "value": ...}.
package tagged

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/npikall/toml-edit/eval"
)

// FromTable returns a value that encoding/json marshals as tagged JSON.
func FromTable(t *eval.Table) map[string]any {
	m := make(map[string]any, len(t.Keys()))
	for _, k := range t.Keys() {
		v, _ := t.Get(k)
		m[k] = fromValue(v)
	}
	return m
}

func fromValue(v any) any {
	switch v := v.(type) {
	case *eval.Table:
		return FromTable(v)
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = fromValue(e)
		}
		return out
	}
	scalar, ok := scalars[reflect.TypeOf(v)]
	if !ok {
		panic(fmt.Sprintf("tagged: unexpected value %T", v))
	}
	return tag(scalar.typ, scalar.format(v))
}

// scalarTag is the tagged JSON type of a scalar and how to write its value.
type scalarTag struct {
	typ    string
	format func(any) string
}

var scalars = map[reflect.Type]scalarTag{
	reflect.TypeFor[string]():             {"string", func(v any) string { return v.(string) }},
	reflect.TypeFor[int64]():              {"integer", func(v any) string { return strconv.FormatInt(v.(int64), 10) }},
	reflect.TypeFor[float64]():            {"float", func(v any) string { return formatFloat(v.(float64)) }},
	reflect.TypeFor[bool]():               {"bool", func(v any) string { return strconv.FormatBool(v.(bool)) }},
	reflect.TypeFor[time.Time]():          {"datetime", func(v any) string { return v.(time.Time).Format(time.RFC3339Nano) }},
	reflect.TypeFor[eval.LocalDateTime](): {"datetime-local", stringer},
	reflect.TypeFor[eval.LocalDate]():     {"date-local", stringer},
	reflect.TypeFor[eval.LocalTime]():     {"time-local", stringer},
}

func stringer(v any) string { return v.(fmt.Stringer).String() }

func formatFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	default:
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}

func tag(typ, value string) map[string]any {
	return map[string]any{"type": typ, "value": value}
}
