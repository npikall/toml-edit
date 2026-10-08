package parser_test

import (
	"testing"

	"github.com/npikall/toml-edit/parser"
	"github.com/stretchr/testify/require"
)

func TestParseSyntaxErrors(t *testing.T) {
	tests := map[string]struct {
		src  string
		want parser.ParseError
	}{
		"missing equals":        {"a 1", parser.ParseError{Line: 1, Col: 3, Msg: `expected "=", found "1"`}},
		"missing value":         {"a =\n", parser.ParseError{Line: 1, Col: 4, Msg: "expected a value, found newline"}},
		"two pairs on one line": {"a = 1 b = 2", parser.ParseError{Line: 1, Col: 7, Msg: `expected a newline, found "b"`}},
		"no key":                {"= 1", parser.ParseError{Line: 1, Col: 1, Msg: `expected a key or table header, found "="`}},
		"trailing dot in key":   {"a.b. = 1", parser.ParseError{Line: 1, Col: 6, Msg: `expected a key, found "="`}},
		"multiline string key":  {"'''k''' = 1", parser.ParseError{Line: 1, Col: 1, Msg: `expected a key or table header, found "'''k'''"`}},
		"unclosed header":       {"[a\nb = 1", parser.ParseError{Line: 1, Col: 3, Msg: `expected "]", found newline`}},
		"half closed aot":       {"[[a]\n", parser.ParseError{Line: 1, Col: 4, Msg: `expected "]]", found "]"`}},
		"junk after header":     {"[a] b\n", parser.ParseError{Line: 1, Col: 5, Msg: `expected a newline, found "b"`}},
		"array missing comma":   {"a = [1 2]", parser.ParseError{Line: 1, Col: 8, Msg: `expected "," or "]", found "2"`}},
		"unclosed array":        {"a = [1,", parser.ParseError{Line: 1, Col: 8, Msg: "expected a value, found end of file"}},
		"array double comma":    {"a = [1,,2]", parser.ParseError{Line: 1, Col: 8, Msg: `expected a value, found ","`}},
		"inline missing comma":  {"a = {x = 1 y = 2}", parser.ParseError{Line: 1, Col: 12, Msg: `expected "," or "}", found "y"`}},
		"stray bracket":         {"a = 1\n]", parser.ParseError{Line: 2, Col: 1, Msg: `expected a key or table header, found "]"`}},
		"bad scalar":            {"a = 1x", parser.ParseError{Line: 1, Col: 5, Msg: `expected a value, found invalid "1x"`}},
		"control char comment":  {"a = 1 # \x01\n", parser.ParseError{Line: 1, Col: 7, Msg: "expected a newline, found invalid \"# \\x01\""}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parser.Parse(tc.src)
			var perr *parser.ParseError
			require.ErrorAs(t, err, &perr)
			require.Equal(t, tc.want, *perr)
		})
	}
}

func TestParseStrict10RejectsTOML11Syntax(t *testing.T) {
	tests := map[string]struct {
		src  string
		want parser.ParseError
	}{
		"hex escape":              {`a = "\x41"`, parser.ParseError{Line: 1, Col: 5, Msg: `expected a value, found invalid "\"\\x41\""`}},
		"esc escape":              {`a = """\e"""`, parser.ParseError{Line: 1, Col: 5, Msg: `expected a value, found invalid "\"\"\"\\e\"\"\""`}},
		"time without seconds":    {"a = 07:32", parser.ParseError{Line: 1, Col: 5, Msg: `expected a value, found invalid "07:32"`}},
		"datetime without secs":   {"a = 1979-05-27T07:32Z", parser.ParseError{Line: 1, Col: 5, Msg: `expected a value, found invalid "1979-05-27T07:32Z"`}},
		"newline in inline table": {"a = {\nb = 1}", parser.ParseError{Line: 1, Col: 6, Msg: "expected a key, found newline"}},
		"comment in inline table": {"a = {b = 1 # c\n}", parser.ParseError{Line: 1, Col: 12, Msg: `expected "," or "}", found "# c"`}},
		"inline trailing comma":   {"a = {b = 1,}", parser.ParseError{Line: 1, Col: 12, Msg: `expected a key, found "}"`}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parser.Parse(tc.src)
			require.NoError(t, err, "valid TOML 1.1")
			_, err = parser.Parse(tc.src, parser.Strict10())
			var perr *parser.ParseError
			require.ErrorAs(t, err, &perr)
			require.Equal(t, tc.want, *perr)
		})
	}
}
