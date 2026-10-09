package tomledit

import (
	"testing"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/token"
	"github.com/stretchr/testify/require"
)

func TestShallowCopyCopiesEveryValueNode(t *testing.T) {
	for _, v := range []ast.Value{&ast.Scalar{Raw: "1"}, &ast.Array{}, &ast.InlineTable{}} {
		c := shallowCopy(v)
		require.Equal(t, v, c)
		require.NotSame(t, v, c)
	}
	require.Nil(t, shallowCopy(nil))
}

func TestCanonicalRejectsUnformattableValue(t *testing.T) {
	_, ok := canonical(struct{}{})
	require.False(t, ok)
}

func TestMutateRestoresDocumentOnInvalidChange(t *testing.T) {
	doc, err := Parse("a = 1\n")
	require.NoError(t, err)
	err = doc.mutate(func() { doc.cst.Root.Body = append(doc.cst.Root.Body, doc.cst.Root.Body[0]) })
	require.Error(t, err)
	require.Equal(t, "a = 1\n", doc.String())
}

func TestMutatePanicsIfDocumentCannotBeRestored(t *testing.T) {
	doc, err := Parse("a = 1\n")
	require.NoError(t, err)
	doc.cst.Trailing = "=" // invalid text the evaluator does not look at
	require.Panics(t, func() {
		_ = doc.mutate(func() { doc.cst.Root.Body = append(doc.cst.Root.Body, doc.cst.Root.Body[0]) })
	})
}

func TestKeyNamesPanicsOnInvalidKey(t *testing.T) {
	key := &ast.Key{Parts: []*ast.KeyPart{{Type: token.BASIC_STRING, Raw: `"\uD800"`}}}
	require.Panics(t, func() { keyNames(key) })
}
