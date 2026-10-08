package parser_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/npikall/toml-edit/parser"

	"github.com/stretchr/testify/require"
	tomltest "github.com/toml-lang/toml-test/v2"
)

// tomlTestFiles returns the toml-test files for TOML 1.1 whose path starts
// with prefix ("valid/" or "invalid/").
func tomlTestFiles(t *testing.T, prefix string) []string {
	t.Helper()
	list, err := fs.ReadFile(tomltest.TestCases(), "files-toml-1.1.0")
	require.NoError(t, err)
	var files []string
	for name := range strings.Lines(string(list)) {
		name = strings.TrimSpace(name)
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".toml") {
			files = append(files, name)
		}
	}
	require.NotEmpty(t, files)
	return files
}

func TestParseRoundTripsTOMLTestValid(t *testing.T) {
	for _, name := range tomlTestFiles(t, "valid/") {
		t.Run(name, func(t *testing.T) {
			src, err := fs.ReadFile(tomltest.TestCases(), name)
			require.NoError(t, err)
			requireRoundTrip(t, string(src))
		})
	}
}

// Invalid files fail either here (syntax) or in the evaluator (semantics).
// Whatever the parser accepts must still round-trip.
func TestParseInvalidTOMLTestFilesRoundTripIfAccepted(t *testing.T) {
	for _, name := range tomlTestFiles(t, "invalid/") {
		t.Run(name, func(t *testing.T) {
			src, err := fs.ReadFile(tomltest.TestCases(), name)
			require.NoError(t, err)
			if doc, err := parser.Parse(string(src)); err == nil {
				require.Equal(t, string(src), doc.String())
			}
		})
	}
}

func FuzzParse(f *testing.F) {
	f.Add("a = [1, {b = 2}]\n[[c]]\n")
	f.Add("[ a . 'b' ] # c\r\nx = { y = [\n1,\n], }\n")
	f.Add("s = \"\"\"\\\n\"\"\" # c")
	f.Fuzz(func(t *testing.T, src string) {
		if doc, err := parser.Parse(src); err == nil {
			require.Equal(t, src, doc.String())
		}
	})
}
