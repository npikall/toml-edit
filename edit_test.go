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

func TestParseStrict10RejectsTOML11Syntax(t *testing.T) {
	src := "point = { x = 1, y = 2, }\n"
	_, err := tomledit.Parse(src)
	require.NoError(t, err)

	_, err = tomledit.Parse(src, tomledit.Strict10())
	var parseErr *parser.ParseError
	require.ErrorAs(t, err, &parseErr)
}

func TestEditsOfStrict10DocumentStayTOML10(t *testing.T) {
	doc, err := tomledit.Parse(config, tomledit.Strict10())
	require.NoError(t, err)
	require.NoError(t, doc.Set([]string{"title"}, "esc \x1b"))
	require.NoError(t, doc.Insert([]string{"server", "point", "z"}, eval.LocalTime{Hour: 7, Minute: 32}))

	_, err = tomledit.Parse(doc.String(), tomledit.Strict10())
	require.NoError(t, err, doc.String())
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

func TestDeleteKeyValue(t *testing.T) {
	doc := parse(t, "a = 1\n# second\nb = 2 # two\nc = 3\n")
	require.NoError(t, doc.Delete("b"))

	_, ok := doc.Get("b")
	require.False(t, ok)
	require.Equal(t, "a = 1\nc = 3\n", doc.String())
}

func TestDeleteErrors(t *testing.T) {
	doc := parse(t, "a = 1\n[[aot]]\nx = 1\n")
	for _, path := range [][]string{nil, {"missing"}, {"a", "b"}, {"aot", "x"}} {
		require.ErrorIs(t, doc.Delete(path...), tomledit.ErrNotFound, path)
	}
	require.Equal(t, "a = 1\n[[aot]]\nx = 1\n", doc.String())
}

func TestInsertAppendsToSection(t *testing.T) {
	doc := parse(t, "a = 1\n\n[t]\n  x = 1 # one\n\n[u]\n")
	require.NoError(t, doc.Insert([]string{"b"}, 2))
	require.NoError(t, doc.Insert([]string{"t", "y"}, "two"))
	require.NoError(t, doc.Insert([]string{"u", "z"}, 3.5))

	y, err := doc.GetString("t", "y")
	require.NoError(t, err)
	require.Equal(t, "two", y)
	require.Equal(t, "a = 1\nb = 2\n\n[t]\n  x = 1 # one\n  y = \"two\"\n\n[u]\nz = 3.5\n", doc.String())
}

func TestInsertUnderDottedTable(t *testing.T) {
	doc := parse(t, "log.level = \"info\"\n[srv]\nhttp.tls.on = true\n")
	require.NoError(t, doc.Insert([]string{"log", "file"}, "x.log"))
	require.NoError(t, doc.Insert([]string{"srv", "http", "tls", "cert"}, "a.pem"))
	require.NoError(t, doc.Insert([]string{"srv", "http", "auth", "user"}, "bob"))

	require.Equal(t, "log.level = \"info\"\nlog.file = \"x.log\"\n[srv]\nhttp.tls.on = true\n"+
		"http.tls.cert = \"a.pem\"\nhttp.auth.user = \"bob\"\n", doc.String())
}

func TestInsertNewTableAtEndOfDocument(t *testing.T) {
	cases := map[string]string{
		"":               "[t]\ny = 2\n",
		"x = 1":          "x = 1\n\n[t]\ny = 2\n",
		"x = 1\n\n\n":    "x = 1\n\n\n[t]\ny = 2\n",
		"x = 1\n# end\n": "x = 1\n# end\n\n[t]\ny = 2\n",
		"x = 1\r\n":      "x = 1\r\n\r\n[t]\r\ny = 2\r\n",
		"[s]\nx = 1 # c": "[s]\nx = 1 # c\n\n[t]\ny = 2\n",
	}
	for in, want := range cases {
		doc := parse(t, in)
		require.NoError(t, doc.Insert([]string{"t", "y"}, 2), in)
		require.Equal(t, want, doc.String(), in)
	}
}

func TestInsertErrors(t *testing.T) {
	const src = "a = 1\nt = { x = 1 }\n[[aot]]\n"
	doc := parse(t, src)
	cases := []struct {
		path []string
		want error
	}{
		{nil, tomledit.ErrNotFound},
		{[]string{"a"}, tomledit.ErrExists},
		{[]string{"t", "x"}, tomledit.ErrExists},
		{[]string{"a", "b"}, tomledit.ErrNotTable},
		{[]string{"aot", "b"}, tomledit.ErrNotTable},
	}
	for _, c := range cases {
		require.ErrorIs(t, doc.Insert(c.path, 2), c.want, c.path)
	}
	require.ErrorIs(t, doc.Insert([]string{"b"}, struct{}{}), format.ErrUnsupported)
	require.Equal(t, src, doc.String())
}

func TestInlineEditsKeepEntryComments(t *testing.T) {
	doc := parse(t, "t = {\n  x = 1,\n  # second\n  y = 2\n}\nu = {\n  x = 1, # c\n}\n")
	require.NoError(t, doc.Delete("t", "x"))
	require.NoError(t, doc.Insert([]string{"u", "y"}, 2))
	require.Equal(t, "t = {\n  # second\n  y = 2\n}\nu = {\n  x = 1, # c\n  y = 2,\n}\n", doc.String())
}

func TestInsertUsesLineEndingOfFirstLine(t *testing.T) {
	doc := parse(t, "a = 1\ns = \"\"\"x\r\ny\"\"\"\n")
	require.NoError(t, doc.Insert([]string{"b"}, 2))
	require.Equal(t, "a = 1\ns = \"\"\"x\r\ny\"\"\"\nb = 2\n", doc.String())
}

func TestSetMultilineArrayEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		value any
		want  string
	}{
		{"empty array", "a = [\n]\n", []any{1, 2}, "a = [\n  1,\n  2,\n]\n"},
		{"only closing on own line", "a = [1, 2\n]\n", []any{1, 2, 3}, "a = [\n  1,\n  2,\n  3,\n]\n"},
		{"inline table element", "a = [\n  {x = 1},\n  2,\n]\n", []any{2, 3}, "a = [\n  2,\n  3,\n]\n"},
		{"nested array keeps comment", "a = [\n  [1], # one\n  2,\n]\n", []any{[]any{1}, 3}, "a = [\n  [1], # one\n  3,\n]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parse(t, tt.src)
			require.NoError(t, doc.Set([]string{"a"}, tt.value))
			require.Equal(t, tt.want, doc.String())
		})
	}
}

func TestInsertEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		src  string
		path []string
		want string
	}{
		{"multi-line inline table with CRLF", "p = {\r\n  x = 1\r\n}\r\n", []string{"p", "y"}, "p = {\r\n  x = 1,\r\n  y = 2\r\n}\r\n"},
		{"after last line without newline", "a = 1", []string{"b"}, "a = 1\nb = 2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parse(t, tt.src)
			require.NoError(t, doc.Insert(tt.path, 2))
			require.Equal(t, tt.want, doc.String())
		})
	}
}
