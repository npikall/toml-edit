package parser_test

import (
	"testing"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/parser"
	"github.com/npikall/toml-edit/token"
	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, src string) *ast.Document {
	t.Helper()
	doc, err := parser.Parse(src)
	require.NoError(t, err)
	return doc
}

func TestParseAttachesCommentsBeforeNodeAsLeading(t *testing.T) {
	doc := mustParse(t, "# header\n\na = 1\n\n  # second\n  b = 2 # same line\n")

	body := doc.Root.Body
	require.Len(t, body, 2)
	require.Equal(t, "# header\n\n", body[0].Leading)
	require.Equal(t, "\n", body[0].Trailing)
	require.Equal(t, "\n  # second\n  ", body[1].Leading)
	require.Equal(t, " # same line\n", body[1].Trailing)
}

func TestParseAttachesCommentsBeforeHeaderToHeader(t *testing.T) {
	doc := mustParse(t, "a = 1\n\n# section b\n[b] # trailing\nx = 1\n# dangling\n")

	require.Len(t, doc.Tables, 1)
	header := doc.Tables[0].Header
	require.Equal(t, "\n# section b\n", header.Leading)
	require.Equal(t, " # trailing\n", header.Trailing)
	require.False(t, header.ArrayOfTables)
	require.Equal(t, "# dangling\n", doc.Trailing)
}

func TestParseKeepsKeyPartsAndDecor(t *testing.T) {
	doc := mustParse(t, "a . \"b c\".'d'  = 1\n")

	kv := doc.Root.Body[0]
	parts := kv.Key.Parts
	require.Len(t, parts, 3)
	require.Equal(t, ast.KeyPart{Type: token.BARE_KEY, Raw: "a", Suffix: " ", Pos: ast.Pos{Line: 1, Col: 1}}, *parts[0])
	require.Equal(t, ast.KeyPart{Type: token.BASIC_STRING, Raw: `"b c"`, Prefix: " ", Pos: ast.Pos{Line: 1, Col: 5}}, *parts[1])
	require.Equal(t, ast.KeyPart{Type: token.LITERAL_STRING, Raw: "'d'", Suffix: "  ", Pos: ast.Pos{Line: 1, Col: 11}}, *parts[2])
	require.Equal(t, &ast.Scalar{Type: token.INTEGER, Raw: "1", Prefix: " ", Pos: ast.Pos{Line: 1, Col: 18}}, kv.Value)
}

func TestParseTableHeaderKeyDecor(t *testing.T) {
	doc := mustParse(t, "[[ a . b ]]\n")

	header := doc.Tables[0].Header
	require.True(t, header.ArrayOfTables)
	require.Equal(t, ast.Decor{Prefix: " ", Suffix: " "}, header.Key.Parts[0].Decor)
	require.Equal(t, ast.Decor{Prefix: " ", Suffix: " "}, header.Key.Parts[1].Decor)
}

func TestParseArrayDecorAndTrailingComma(t *testing.T) {
	doc := mustParse(t, "a = [ 1 ,\n  2, # two\n]\nb = [1, 2]\nc = [ ]\n")

	arr := doc.Root.Body[0].Value.(*ast.Array)
	require.Len(t, arr.Items, 2)
	require.Equal(t, ast.Decor{Prefix: " ", Suffix: " "}, *ast.ValueDecor(arr.Items[0].Value))
	require.Equal(t, "\n", arr.Items[0].AfterComma)
	require.Equal(t, ast.Decor{Prefix: "  "}, *ast.ValueDecor(arr.Items[1].Value))
	require.Equal(t, " # two\n", arr.Items[1].AfterComma)
	require.True(t, arr.TrailingComma)
	require.Empty(t, arr.Trailing)

	arr = doc.Root.Body[1].Value.(*ast.Array)
	require.Empty(t, arr.Items[0].AfterComma)
	require.Equal(t, ast.Decor{Prefix: " "}, *ast.ValueDecor(arr.Items[1].Value))
	require.False(t, arr.TrailingComma)
	require.Empty(t, arr.Trailing)

	arr = doc.Root.Body[2].Value.(*ast.Array)
	require.Empty(t, arr.Items)
	require.Equal(t, " ", arr.Trailing)
}

// A comment after a comma belongs to the element before the comma, so
// deleting that element takes its comment along.
func TestParseArrayCommentAfterCommaBelongsToPrecedingItem(t *testing.T) {
	doc := mustParse(t, "a = [\n  1, # one\n  # about two\n  2 # two\n]\n")

	arr := doc.Root.Body[0].Value.(*ast.Array)
	require.Len(t, arr.Items, 2)
	require.Equal(t, " # one\n", arr.Items[0].AfterComma)
	require.Equal(t, ast.Decor{Prefix: "  # about two\n  ", Suffix: " # two\n"}, *ast.ValueDecor(arr.Items[1].Value))
	require.False(t, arr.TrailingComma)
}

func TestParseInlineTableEntries(t *testing.T) {
	doc := mustParse(t, "t = { x = 1 , y.z = 'a', # last\n}\n")

	table := doc.Root.Body[0].Value.(*ast.InlineTable)
	require.Len(t, table.Entries, 2)
	require.Equal(t, " ", table.Entries[0].KeyValue.Leading)
	require.Equal(t, " ", table.Entries[0].KeyValue.Trailing)
	require.Empty(t, table.Entries[0].AfterComma)
	require.Len(t, table.Entries[1].KeyValue.Key.Parts, 2)
	require.Equal(t, " # last\n", table.Entries[1].AfterComma)
	require.True(t, table.TrailingComma)
	require.Empty(t, table.Trailing)
}
