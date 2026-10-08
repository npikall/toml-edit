package format_test

import (
	"math"
	"testing"
	"time"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/format"
	"github.com/npikall/toml-edit/parser"
	"github.com/stretchr/testify/require"
)

// render formats v as the value of "v = ...".
func render(t *testing.T, v any) string {
	t.Helper()
	value, err := format.Value(v)
	require.NoError(t, err)
	doc := &ast.Document{Root: &ast.Table{Body: []*ast.KeyValue{{
		Key:   &ast.Key{Parts: []*ast.KeyPart{format.Key("v")}},
		Value: value,
	}}}}
	return doc.String()
}

// decode parses src and returns the value of its key "v".
func decode(t *testing.T, src string) any {
	t.Helper()
	doc, err := parser.Parse(src)
	require.NoError(t, err, src)
	table, err := eval.Evaluate(doc)
	require.NoError(t, err, src)
	v, ok := table.Get("v")
	require.True(t, ok)
	return v
}

func TestValueRendersCanonicalText(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"hello", `v="hello"`},
		{"a\"b\\c\n\t\x1b\x7f", `v="a\"b\\c\n\t\e\u007F"`},
		{true, "v=true"},
		{42, "v=42"},
		{int8(-7), "v=-7"},
		{uint16(7), "v=7"},
		{1.5, "v=1.5"},
		{100.0, "v=100.0"},
		{float32(0.1), "v=0.1"},
		{1e300, "v=1e+300"},
		{math.Inf(1), "v=inf"},
		{math.Inf(-1), "v=-inf"},
		{math.NaN(), "v=nan"},
		{time.Date(1979, 5, 27, 7, 32, 0, 0, time.UTC), "v=1979-05-27T07:32:00Z"},
		{time.Date(1979, 5, 27, 0, 32, 0, 999999000, time.FixedZone("", -7*3600)), "v=1979-05-27T00:32:00.999999-07:00"},
		{eval.LocalDate{Year: 1979, Month: 5, Day: 27}, "v=1979-05-27"},
		{eval.LocalTime{Hour: 7, Minute: 32}, "v=07:32:00"},
		{eval.LocalDateTime{Date: eval.LocalDate{Year: 1979, Month: 5, Day: 27}, Time: eval.LocalTime{Hour: 7}}, "v=1979-05-27T07:00:00"},
		{[]any{}, "v=[]"},
		{[]int{1, 2, 3}, "v=[1, 2, 3]"},
		{[]any{"a", []any{true}}, `v=["a", [true]]`},
		{map[string]any{}, "v={}"},
		{map[string]any{"b": 2, "a": "x", "needs quote": 1}, `v={ a = "x", b = 2, "needs quote" = 1 }`},
	}
	for _, c := range cases {
		require.Equal(t, c.want, render(t, c.in), "%#v", c.in)
	}
}

func TestValueRoundTripsThroughParser(t *testing.T) {
	for _, in := range []any{
		"", "ünïcödé ✓", "\x00\x01\x1f\b\f\r", `C:\path`, "'single'",
		int64(math.MinInt64), int64(math.MaxInt64), uint64(math.MaxInt64),
		0.1, -0.0, 1e-7, 123456789.125, math.MaxFloat64, math.SmallestNonzeroFloat64,
	} {
		v := decode(t, render(t, in))
		switch in := in.(type) {
		case uint64:
			require.Equal(t, int64(in), v)
		default:
			require.Equal(t, in, v)
		}
	}
}

func TestValueRejectsUnsupported(t *testing.T) {
	for _, in := range []any{
		nil, "\xff", uint64(math.MaxInt64) + 1,
		struct{}{},
		map[int]any{1: 1},
		[]any{make(chan int)},
		time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.FixedZone("LMT", 561)),
	} {
		_, err := format.Value(in)
		require.Error(t, err, "%#v", in)
	}
}

func TestKeyQuotesOnlyWhenNeeded(t *testing.T) {
	cases := map[string]string{
		"port":   "port",
		"a-b_C9": "a-b_C9",
		"":       `""`,
		"a.b":    `"a.b"`,
		"ключ":   `"ключ"`,
		`q"`:     `"q\""`,
	}
	for in, want := range cases {
		require.Equal(t, want, format.Key(in).Raw, in)
	}
}
