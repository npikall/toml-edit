package tomledit_test

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"

	tomledit "github.com/npikall/toml-edit"
	"github.com/npikall/toml-edit/eval"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite the golden .out.toml files")

// editCases maps a golden file name in testdata/edit to the edit applied to
// NAME.in.toml; the result must equal NAME.out.toml byte for byte.
var editCases = map[string]func(*tomledit.Document) error{
	"set-scalar": func(d *tomledit.Document) error {
		return d.Set([]string{"server", "port"}, 9090)
	},
	"set-quoted-dotted-key": func(d *tomledit.Document) error {
		return errors.Join(
			d.Set([]string{"my key"}, "new"),
			d.Set([]string{"site", "google.com", "enabled"}, true),
		)
	},
	"set-inline-table-entry": func(d *tomledit.Document) error {
		return d.Set([]string{"point", "x"}, 10)
	},
	"set-multiline-inline-table": func(d *tomledit.Document) error {
		return d.Set([]string{"owner", "name"}, "Tom Preston-Werner")
	},
	"set-change-type": func(d *tomledit.Document) error {
		return errors.Join(
			d.Set([]string{"deps", "alpha"}, map[string]any{"version": "2.0", "git": "https://example.com/alpha"}),
			d.Set([]string{"deps", "beta"}, []string{"z"}),
		)
	},
	"set-multiline-array": func(d *tomledit.Document) error {
		return errors.Join(
			d.Set([]string{"tool", "gotpm", "dependencies"}, []string{"@gotpm/a:1.0.0", "@gotpm/b:2.0.0"}),
			d.Set([]string{"tool", "gotpm", "fonts"}, []string{"Roboto", "Inter"}),
		)
	},
	"set-multiline-array-comments": func(d *tomledit.Document) error {
		return errors.Join(
			d.Set([]string{"deps"}, []string{"c", "a", "d"}),
			d.Set([]string{"more"}, []string{"y"}),
		)
	},
	"set-multiline-array-edges": func(d *tomledit.Document) error {
		return errors.Join(
			d.Set([]string{"a"}, []int{1, 2}),
			d.Set([]string{"b"}, []int{}),
			d.Set([]string{"t", "x"}, []int{3}),
			d.Set([]string{"m"}, []int{1, 4}),
		)
	},
	"set-multiline-array-mixed": func(d *tomledit.Document) error {
		return errors.Join(
			d.Set([]string{"a"}, []int{1, 2, 3, 4}),
			d.Set([]string{"b"}, []int{2, 1}),
		)
	},
	"set-singleline-array": func(d *tomledit.Document) error {
		return d.Set([]string{"tags"}, []string{"a", "b", "c"})
	},
	"set-crlf": func(d *tomledit.Document) error {
		return d.Set([]string{"b"}, eval.LocalDate{Year: 2025, Month: 1, Day: 31})
	},
	"delete-table": func(d *tomledit.Document) error {
		return errors.Join(d.Delete("server"), d.Delete("log"), d.Delete("plugin"))
	},
	"delete-inline-entry": func(d *tomledit.Document) error {
		return errors.Join(
			d.Delete("first", "x"),
			d.Delete("last", "y"),
			d.Delete("middle", "y"),
			d.Delete("only", "x"),
			d.Delete("trailing", "y"),
			d.Delete("dotted", "a"),
			d.Delete("multi", "y"),
			d.Delete("multi-trailing", "x"),
		)
	},
	"insert-table": func(d *tomledit.Document) error {
		return errors.Join(
			d.Insert([]string{"database", "url"}, "pg"),
			d.Insert([]string{"server", "tls", "cert"}, "a.pem"),
			d.Insert([]string{"a", "name"}, "A"),
		)
	},
	"insert-multiline-array": func(d *tomledit.Document) error {
		return errors.Join(
			d.Insert([]string{"package", "point", "y"}, []int{1}, tomledit.Multiline()),
			d.Insert([]string{"package", "keywords"}, []string{"x"}, tomledit.Multiline()),
			d.Insert([]string{"tool", "gotpm", "dependencies"}, []string{"a", "b"}, tomledit.Multiline()),
		)
	},
	"insert-inline-entry": func(d *tomledit.Document) error {
		return errors.Join(
			d.Insert([]string{"empty", "k"}, 1),
			d.Insert([]string{"spaced", "y"}, 2),
			d.Insert([]string{"tight", "y"}, 2),
			d.Insert([]string{"trailing", "y"}, 2),
			d.Insert([]string{"nested", "a", "c"}, 2),
			d.Insert([]string{"nested", "c", "e"}, 3),
			d.Insert([]string{"nested", "f", "g"}, 4),
			d.Insert([]string{"multi", "z"}, 3),
			d.Insert([]string{"multi-trailing", "y"}, 2),
		)
	},
}

func TestEditGolden(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "edit", "*.in.toml"))
	require.NoError(t, err)
	require.Len(t, inputs, len(editCases), "every golden input needs an edit case and vice versa")
	for name, edit := range editCases {
		t.Run(name, func(t *testing.T) {
			base := filepath.Join("testdata", "edit", name)
			in, err := os.ReadFile(base + ".in.toml")
			require.NoError(t, err)
			doc, err := tomledit.Parse(string(in))
			require.NoError(t, err)
			require.NoError(t, edit(doc))
			got := doc.String()

			// The output must itself be valid TOML.
			_, err = tomledit.Parse(got)
			require.NoError(t, err)

			if *update {
				require.NoError(t, os.WriteFile(base+".out.toml", []byte(got), 0o600))
			}
			want, err := os.ReadFile(base + ".out.toml")
			require.NoError(t, err)
			require.Equal(t, string(want), got)
		})
	}
}
