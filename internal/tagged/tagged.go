// Package tagged converts evaluated TOML into toml-test's tagged JSON form,
// where every scalar is {"type": ..., "value": ...}.
package tagged

import (
	"fmt"
	"math"
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
	case string:
		return tag("string", v)
	case int64:
		return tag("integer", strconv.FormatInt(v, 10))
	case float64:
		return tag("float", formatFloat(v))
	case bool:
		return tag("bool", strconv.FormatBool(v))
	case time.Time:
		return tag("datetime", v.Format(time.RFC3339Nano))
	case eval.LocalDateTime:
		return tag("datetime-local", v.String())
	case eval.LocalDate:
		return tag("date-local", v.String())
	case eval.LocalTime:
		return tag("time-local", v.String())
	default:
		panic(fmt.Sprintf("tagged: unexpected value %T", v))
	}
}

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
