package eval

import (
	"testing"

	"github.com/npikall/toml-edit/ast"
	"github.com/stretchr/testify/require"
)

// unknownValue is an ast.Value of a type the evaluator does not know.
type unknownValue struct{ *ast.Scalar }

func TestDecodeValuePanicsOnUnknownNode(t *testing.T) {
	require.Panics(t, func() { _, _ = decodeValue(unknownValue{&ast.Scalar{}}) })
}
