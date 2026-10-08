package eval_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/parser"
	"github.com/stretchr/testify/require"
)

func evaluate(t *testing.T, src string) (*eval.Table, error) {
	t.Helper()
	doc, err := parser.Parse(src)
	require.NoError(t, err)
	table, err := eval.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}
	return table, nil
}

// toMap converts a table tree into plain maps so tests can compare it with a
// literal.
func toMap(t *eval.Table) map[string]any {
	m := map[string]any{}
	for _, k := range t.Keys() {
		v, _ := t.Get(k)
		m[k] = plain(v)
	}
	return m
}

func plain(v any) any {
	switch v := v.(type) {
	case *eval.Table:
		return toMap(v)
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = plain(e)
		}
		return out
	default:
		return v
	}
}

func requireEval(t *testing.T, src string, want map[string]any) {
	t.Helper()
	table, err := evaluate(t, src)
	require.NoError(t, err)
	require.Equal(t, want, toMap(table))
}

func TestEvaluateBasicScalars(t *testing.T) {
	requireEval(t, "s = 'lit'\nb = true\nf = false\ni = 42\n", map[string]any{
		"s": "lit", "b": true, "f": false, "i": int64(42),
	})
}

func TestEvaluateIntegers(t *testing.T) {
	requireEval(t, "a = +1_000\nb = -17\nc = 0xDEAD_beef\nd = 0o755\ne = 0b1101\nf = 9223372036854775807\ng = -9223372036854775808\n", map[string]any{
		"a": int64(1000), "b": int64(-17), "c": int64(0xdeadbeef), "d": int64(0o755), "e": int64(0b1101),
		"f": int64(9223372036854775807), "g": int64(-9223372036854775808),
	})
}

func TestEvaluateIntegerOverflow(t *testing.T) {
	for _, src := range []string{"a = 9223372036854775808", "a = -9223372036854775809", "a = 0x1_0000_0000_0000_0000"} {
		_, err := evaluate(t, src)
		var evalErr *eval.Error
		require.ErrorAs(t, err, &evalErr, src)
		require.Equal(t, eval.Error{Line: 1, Col: 5, Msg: "integer out of range"}, *evalErr, src)
	}
}

func TestEvaluateFloats(t *testing.T) {
	requireEval(t, "a = +1.5\nb = -0.01\nc = 5e+22\nd = 6.626e-34\ne = 224_617.445_991\nf = inf\ng = -inf\n", map[string]any{
		"a": 1.5, "b": -0.01, "c": 5e+22, "d": 6.626e-34, "e": 224617.445991,
		"f": math.Inf(1), "g": math.Inf(-1),
	})

	table, err := evaluate(t, "n = nan\nm = -nan\n")
	require.NoError(t, err)
	n, _ := table.Get("n")
	require.True(t, math.IsNaN(n.(float64)))
	m, _ := table.Get("m")
	require.True(t, math.IsNaN(m.(float64)))
}

func TestEvaluateFloatOverflow(t *testing.T) {
	_, err := evaluate(t, "a = 1e400")
	var evalErr *eval.Error
	require.ErrorAs(t, err, &evalErr)
	require.Equal(t, eval.Error{Line: 1, Col: 5, Msg: "float out of range"}, *evalErr)
}

func TestEvaluateStrings(t *testing.T) {
	src := `basic = "tab\there \"q\" \\ \u00e9 \U0001F600 \x41 \e."
literal = 'C:\path'
ml = """
line1
line2"""
trimmed = """\
    one \
    two"""
mllit = '''
raw \n'''
`
	requireEval(t, src, map[string]any{
		"basic":   "tab\there \"q\" \\ é 😀 A \x1b.",
		"literal": `C:\path`,
		"ml":      "line1\nline2",
		"trimmed": "one two",
		"mllit":   `raw \n`,
	})
}

func TestEvaluateInvalidUnicodeEscape(t *testing.T) {
	for _, src := range []string{`a = "\uD800"`, `a = "\U00110000"`} {
		_, err := evaluate(t, src)
		var evalErr *eval.Error
		require.ErrorAs(t, err, &evalErr, src)
		require.Equal(t, 5, evalErr.Col, src)
	}
}

func TestEvaluateDateTimes(t *testing.T) {
	src := "odt = 1979-05-27T07:32:00.5-07:00\nutc = 1979-05-27 07:32Z\nldt = 1979-05-27t07:32:01.123456789123\nld = 2000-02-29\nlt = 07:32\n"
	requireEval(t, src, map[string]any{
		"odt": time.Date(1979, 5, 27, 7, 32, 0, 500000000, time.FixedZone("", -7*60*60)),
		"utc": time.Date(1979, 5, 27, 7, 32, 0, 0, time.UTC),
		"ldt": eval.LocalDateTime{
			Date: eval.LocalDate{Year: 1979, Month: 5, Day: 27},
			Time: eval.LocalTime{Hour: 7, Minute: 32, Second: 1, Nanosecond: 123456789},
		},
		"ld": eval.LocalDate{Year: 2000, Month: 2, Day: 29},
		"lt": eval.LocalTime{Hour: 7, Minute: 32},
	})
}

func TestEvaluateDateTimeOutOfRange(t *testing.T) {
	for _, src := range []string{
		"a = 2001-13-01", "a = 2001-00-01", "a = 2100-02-29", "a = 2001-04-31", "a = 2001-01-00",
		"a = 24:00:00", "a = 23:60:00", "a = 23:59:61",
		"a = 2001-01-01T00:00:00+24:00", "a = 2001-01-01T00:00:00+01:60",
	} {
		_, err := evaluate(t, src)
		var evalErr *eval.Error
		require.ErrorAs(t, err, &evalErr, src)
		require.Equal(t, 5, evalErr.Col, src)
	}
}

func TestEvaluateArraysAndInlineTables(t *testing.T) {
	src := "arr = [1, 'two', [3.0], {x = 1}]\nempty = []\npoint = { x = 1, y.z = 2, 'q k' = {} }\n"
	requireEval(t, src, map[string]any{
		"arr":   []any{int64(1), "two", []any{3.0}, map[string]any{"x": int64(1)}},
		"empty": []any{},
		"point": map[string]any{"x": int64(1), "y": map[string]any{"z": int64(2)}, "q k": map[string]any{}},
	})
}

func TestEvaluateTables(t *testing.T) {
	src := `top = 1
[a.b]
x = 1
[a]
y = 2
[fruit]
apple.color = "red"
[fruit.apple.texture]
smooth = true
[[products]]
name = "Hammer"
[products.dims]
w = 1
[[products]]
[[products]]
name = "Nail"
`
	requireEval(t, src, map[string]any{
		"top": int64(1),
		"a":   map[string]any{"b": map[string]any{"x": int64(1)}, "y": int64(2)},
		"fruit": map[string]any{"apple": map[string]any{
			"color": "red", "texture": map[string]any{"smooth": true},
		}},
		"products": []any{
			map[string]any{"name": "Hammer", "dims": map[string]any{"w": int64(1)}},
			map[string]any{},
			map[string]any{"name": "Nail"},
		},
	})
}

func TestEvaluateSemanticErrors(t *testing.T) {
	tests := map[string]struct {
		src  string
		want eval.Error
	}{
		"duplicate key":                     {"a = 1\na = 2", eval.Error{Line: 2, Col: 1, Msg: `duplicate key "a"`}},
		"duplicate quoted key":              {"a = 1\n\"a\" = 2", eval.Error{Line: 2, Col: 1, Msg: `duplicate key "a"`}},
		"duplicate in inline":               {"t = {a = 1, a = 2}", eval.Error{Line: 1, Col: 13, Msg: `duplicate key "a"`}},
		"table redefined":                   {"[a]\n[a]", eval.Error{Line: 2, Col: 2, Msg: `table "a" already defined`}},
		"header after dotted":               {"a.b = 1\n[a]", eval.Error{Line: 2, Col: 2, Msg: `table "a" already defined`}},
		"dotted into header":                {"[a.b]\n[a]\nb.c = 1", eval.Error{Line: 3, Col: 1, Msg: `cannot add keys to table "b" with a dotted key`}},
		"header after dotted into implicit": {"[a.b.c]\n[a]\nb.d = 1\n[a.b]", eval.Error{Line: 4, Col: 4, Msg: `table "b" already defined`}},
		"extend inline by header":           {"a = {}\n[a.b]", eval.Error{Line: 2, Col: 2, Msg: `cannot extend inline table "a"`}},
		"extend inline by dotted":           {"a = {}\na.b = 1", eval.Error{Line: 2, Col: 1, Msg: `cannot add keys to table "a" with a dotted key`}},
		"aot over static array":             {"a = []\n[[a]]", eval.Error{Line: 2, Col: 3, Msg: `cannot define array of tables "a": key already defined`}},
		"table over aot":                    {"[[a]]\n[a]", eval.Error{Line: 2, Col: 2, Msg: `table "a" already defined`}},
		"header through value":              {"a = 1\n[a.b]", eval.Error{Line: 2, Col: 2, Msg: `key "a" is already defined as a value`}},
		"header through array":              {"a = [{}]\n[a.b]", eval.Error{Line: 2, Col: 2, Msg: `cannot extend static array "a"`}},
		"dotted through value":              {"a = 1\na.b = 2", eval.Error{Line: 2, Col: 1, Msg: `key "a" is already defined as a value`}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := evaluate(t, tc.src)
			var evalErr *eval.Error
			require.ErrorAs(t, err, &evalErr)
			require.Equal(t, tc.want, *evalErr)
		})
	}
}

func FuzzEvaluate(f *testing.F) {
	f.Add("a.b = 1\n[c]\nd = [1, {e = 2}]\n[[f]]\n[f.g]\n")
	f.Add("s = \"\\u00e9\\x41\\\n  x\"\nt = 1979-05-27T07:32:00.123-07:00\n")
	f.Add("a = 0x_1\nb = 1e400\nc = 24:00\n")
	f.Fuzz(func(t *testing.T, src string) {
		doc, err := parser.Parse(src)
		if err != nil {
			return
		}
		_, _ = eval.Evaluate(doc) // must not panic
	})
}

// A table that only exists as the parent of a header has not been defined
// yet, so dotted keys may add to it (as BurntSushi/toml does).
func TestEvaluateDottedKeyExtendsImplicitTable(t *testing.T) {
	requireEval(t, "[a.b.c]\n[a]\nb.d = 1\n", map[string]any{
		"a": map[string]any{"b": map[string]any{"c": map[string]any{}, "d": int64(1)}},
	})
}
