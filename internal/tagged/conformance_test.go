package tagged_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/internal/tagged"
	"github.com/npikall/toml-edit/parser"
	"github.com/stretchr/testify/require"
	tomltest "github.com/toml-lang/toml-test/v2"
)

// tomlTestFiles returns the toml-test files for a TOML version ("1.0.0" or
// "1.1.0") whose path starts with prefix ("valid/" or "invalid/").
func tomlTestFiles(t *testing.T, version, prefix string) []string {
	t.Helper()
	list, err := fs.ReadFile(tomltest.TestCases(), "files-toml-"+version)
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

func decode(src string, opts ...parser.Option) (*eval.Table, error) {
	doc, err := parser.Parse(src, opts...)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	table, err := eval.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}
	return table, nil
}

// versions lists each TOML version with the parse options that select it.
var versions = []struct {
	name string
	opts []parser.Option
}{
	{"1.1.0", nil},
	{"1.0.0", []parser.Option{parser.Strict10()}},
}

func TestDecodesTOMLTestValid(t *testing.T) {
	for _, version := range versions {
		for _, name := range tomlTestFiles(t, version.name, "valid/") {
			t.Run(version.name+"/"+name, func(t *testing.T) {
				src, err := fs.ReadFile(tomltest.TestCases(), name)
				require.NoError(t, err)
				wantJSON, err := fs.ReadFile(tomltest.TestCases(), strings.TrimSuffix(name, ".toml")+".json")
				require.NoError(t, err)

				table, err := decode(string(src), version.opts...)
				require.NoError(t, err)

				// Round-trip through JSON so types match what CompareJSON expects.
				haveJSON, err := json.Marshal(tagged.FromTable(table))
				require.NoError(t, err)
				var want, have any
				require.NoError(t, json.Unmarshal(wantJSON, &want))
				require.NoError(t, json.Unmarshal(haveJSON, &have))
				result := tomltest.Test{Path: name}.CompareJSON(want, have)
				require.False(t, result.Failed(), "%s\ninput:\n%s", result.Failure, src)
			})
		}
	}
}

func TestRejectsTOMLTestInvalid(t *testing.T) {
	for _, version := range versions {
		for _, name := range tomlTestFiles(t, version.name, "invalid/") {
			t.Run(version.name+"/"+name, func(t *testing.T) {
				src, err := fs.ReadFile(tomltest.TestCases(), name)
				require.NoError(t, err)
				_, err = decode(string(src), version.opts...)
				require.Error(t, err, "input:\n%s", src)
			})
		}
	}
}
