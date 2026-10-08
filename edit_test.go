package tomledit_test

import (
	"testing"

	tomledit "github.com/npikall/toml-edit"
	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/format"
	"github.com/npikall/toml-edit/parser"
	"github.com/stretchr/testify/require"
)

const config = `# app config
title = "demo"

[server]
host = "localhost" # bind address
port = 8080
ratio = 0.5
debug = true
point = { x = 1, y = 2 }
`

func parse(t *testing.T, src string) *tomledit.Document {
	t.Helper()
	doc, err := tomledit.Parse(src)
	require.NoError(t, err)
	return doc
}

func TestParseRoundTrips(t *testing.T) {
	require.Equal(t, config, parse(t, config).String())
}

func TestParseReportsSyntaxAndSemanticErrors(t *testing.T) {
	_, err := tomledit.Parse("a = ")
	var parseErr *parser.ParseError
	require.ErrorAs(t, err, &parseErr)

	_, err = tomledit.Parse("a = 1\na = 2\n")
	var evalErr *eval.Error
	require.ErrorAs(t, err, &evalErr)
}

func TestGet(t *testing.T) {
	doc := parse(t, config)

	v, ok := doc.Get("title")
	require.True(t, ok)
	require.Equal(t, "demo", v)

	v, ok = doc.Get("server", "point", "y")
	require.True(t, ok)
	require.Equal(t, int64(2), v)

	v, ok = doc.Get("server")
	require.True(t, ok)
	require.IsType(t, &eval.Table{}, v)

	for _, path := range [][]string{{"missing"}, {"server", "missing"}, {"title", "x"}} {
		_, ok = doc.Get(path...)
		require.False(t, ok, path)
	}
}

func TestTypedGetters(t *testing.T) {
	doc := parse(t, config)

	port, err := doc.GetInt("server", "port")
	require.NoError(t, err)
	require.Equal(t, int64(8080), port)

	ratio, err := doc.GetFloat("server", "ratio")
	require.NoError(t, err)
	require.InDelta(t, 0.5, ratio, 0)

	host, err := doc.GetString("server", "host")
	require.NoError(t, err)
	require.Equal(t, "localhost", host)

	debug, err := doc.GetBool("server", "debug")
	require.NoError(t, err)
	require.True(t, debug)

	_, err = doc.GetInt("server", "host")
	require.ErrorIs(t, err, tomledit.ErrType)
	require.EqualError(t, err, `server.host: wrong value type: got string, want int64`)

	_, err = doc.GetString("server", "nope")
	require.ErrorIs(t, err, tomledit.ErrNotFound)
}

func TestSetUpdatesValueAndText(t *testing.T) {
	doc := parse(t, "a = 1 # one\n")
	require.NoError(t, doc.Set([]string{"a"}, "two"))

	v, err := doc.GetString("a")
	require.NoError(t, err)
	require.Equal(t, "two", v)
	require.Equal(t, "a = \"two\" # one\n", doc.String())
}

func TestSetErrors(t *testing.T) {
	doc := parse(t, "a.b = 1\n[t]\nx = 1\n[[aot]]\n")
	cases := []struct {
		path []string
		want error
	}{
		{nil, tomledit.ErrNotFound},
		{[]string{"missing"}, tomledit.ErrNotFound},
		{[]string{"t", "missing"}, tomledit.ErrNotFound},
		{[]string{"t", "x", "y"}, tomledit.ErrNotFound},
		{[]string{"a"}, tomledit.ErrNotValue},
		{[]string{"t"}, tomledit.ErrNotValue},
		{[]string{"aot"}, tomledit.ErrNotValue},
	}
	for _, c := range cases {
		require.ErrorIs(t, doc.Set(c.path, 2), c.want, c.path)
	}
	require.ErrorIs(t, doc.Set([]string{"t", "x"}, struct{}{}), format.ErrUnsupported)
	require.Equal(t, "a.b = 1\n[t]\nx = 1\n[[aot]]\n", doc.String())
}

func TestSetRebuildsSemanticTree(t *testing.T) {
	doc := parse(t, "x = 1\n")
	require.NoError(t, doc.Set([]string{"x"}, map[string]any{"y": []int{1, 2}}))

	v, ok := doc.Get("x", "y")
	require.True(t, ok)
	require.Equal(t, []any{int64(1), int64(2)}, v)

	require.NoError(t, doc.Set([]string{"x", "y"}, "z"))
	require.Equal(t, "x = { y = \"z\" }\n", doc.String())
}
