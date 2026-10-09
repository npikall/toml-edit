package ast_test

import (
	"testing"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/parser"
	"github.com/stretchr/testify/require"
)

func TestDocumentStringRoundTrips(t *testing.T) {
	src := `# leading comment
title = "demo" # trailing
a.b . c = 1

[ table . "quoted" ] # header comment
arr = [ 1, [2, 3], # after comma
  4, ]
empty = []
point = { x = 1, y = { z = 2 }, }
none = {}

[[aot]]
k = 'v'
# document trailing
`
	doc, err := parser.Parse(src)
	require.NoError(t, err)
	require.Equal(t, src, doc.String())
}

func TestRawOmitsDecor(t *testing.T) {
	doc, err := parser.Parse("a =  [ 1, { x = 2 } ]  \n")
	require.NoError(t, err)
	v := doc.Root.Body[0].Value
	require.Equal(t, "[ 1, { x = 2 } ]", ast.Raw(v))
	require.Equal(t, &ast.Decor{Prefix: "  ", Suffix: ""}, ast.ValueDecor(v))
}
