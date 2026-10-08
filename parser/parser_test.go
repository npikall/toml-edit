package parser_test

import (
	"testing"

	"github.com/npikall/toml-edit/parser"
	"github.com/stretchr/testify/require"
)

func requireRoundTrip(t *testing.T, src string) {
	t.Helper()
	doc, err := parser.Parse(src)
	require.NoError(t, err)
	require.Equal(t, src, doc.String())
}

func TestParseRoundTrip(t *testing.T) {
	tests := map[string]string{
		"empty":                  "",
		"key value":              "title = \"TOML\" # the name\n",
		"dotted key":             "a . \"b\" .'c'  =1",
		"table":                  "[a]\nx = 1\n\n# next\n[ b . c ] # c\ny = 2\n",
		"array":                  "a = [ 1, [2,3], \"x\" ]\nb = []\nc = [ ]\n",
		"multiline array":        "a = [\n  1, # one\n  2,\n  # end\n]\n",
		"inline table":           "a = { x = 1, y.z = [ {} ] }\nb = {}\nc = { }\n",
		"multiline inline table": "a = {\n  x = 1, # one\n  y = 2,\n}\n",
		"array of tables":        "[[a]]\nx = 1\n[[ a ]]\r\nx = 2\n",
	}
	for name, src := range tests {
		t.Run(name, func(t *testing.T) { requireRoundTrip(t, src) })
	}
}
