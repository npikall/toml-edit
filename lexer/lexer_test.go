package lexer_test

import (
	"strings"
	"testing"

	. "github.com/npikall/toml-edit/lexer"
	"github.com/npikall/toml-edit/token"
	"github.com/stretchr/testify/require"
)

type wantToken struct {
	wantType    token.TokenType
	wantLiteral string
}

func requireTokens(t *testing.T, input string, want []wantToken) {
	t.Helper()
	l := New(input)
	for i, w := range want {
		tok := l.NextToken()
		require.Equal(t, w.wantType, tok.Type, "tests[%d] - wrong type (literal %q)", i, tok.Literal)
		require.Equal(t, w.wantLiteral, tok.Literal, "tests[%d] - wrong literal", i)
	}
	require.Equal(t, token.TokenType(token.EOF), l.NextToken().Type, "expected EOF after all tokens")
}

func TestNextTokenKeyValue(t *testing.T) {
	input := "title = \"TOML\" # the name\n"
	requireTokens(t, input, []wantToken{
		{token.BARE_KEY, "title"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.BASIC_STRING, `"TOML"`},
		{token.WHITESPACE, " "},
		{token.COMMENT, "# the name"},
		{token.NEWLINE, "\n"},
	})
}

func TestNextTokenPositions(t *testing.T) {
	input := "a = 1\r\n  b = 2\n"
	want := []struct {
		literal   string
		line, col int
	}{
		{"a", 1, 1},
		{" ", 1, 2},
		{"=", 1, 3},
		{" ", 1, 4},
		{"1", 1, 5},
		{"\r\n", 1, 6},
		{"  ", 2, 1},
		{"b", 2, 3},
		{" ", 2, 4},
		{"=", 2, 5},
		{" ", 2, 6},
		{"2", 2, 7},
		{"\n", 2, 8},
		{"", 3, 1},
	}
	l := New(input)
	for i, w := range want {
		tok := l.NextToken()
		require.Equal(t, w.literal, tok.Literal, "tests[%d] - wrong literal", i)
		require.Equal(t, w.line, tok.Line, "tests[%d] - wrong line", i)
		require.Equal(t, w.col, tok.Col, "tests[%d] - wrong col", i)
	}
}

func TestNextTokenKeyVersusValueContext(t *testing.T) {
	input := "1979-05-27 = 1979-05-27\ntrue = true\n1.5 = 1.5\n"
	requireTokens(t, input, []wantToken{
		{token.BARE_KEY, "1979-05-27"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.LOCAL_DATE, "1979-05-27"},
		{token.NEWLINE, "\n"},
		{token.BARE_KEY, "true"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.BOOL, "true"},
		{token.NEWLINE, "\n"},
		{token.BARE_KEY, "1"},
		{token.DOT, "."},
		{token.BARE_KEY, "5"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.FLOAT, "1.5"},
		{token.NEWLINE, "\n"},
	})
}

// valueToken lexes `v = <value>` and returns the token for the value.
func valueToken(t *testing.T, value string) token.Token {
	t.Helper()
	l := New("v = " + value)
	for range 4 { // BARE_KEY, WHITESPACE, EQUALS, WHITESPACE
		l.NextToken()
	}
	return l.NextToken()
}

func TestNextTokenScalars(t *testing.T) {
	tests := []struct {
		input    string
		wantType token.TokenType
	}{
		{"true", token.BOOL},
		{"false", token.BOOL},
		{"truex", token.ILLEGAL},

		{"0", token.INTEGER},
		{"+99", token.INTEGER},
		{"-17", token.INTEGER},
		{"1_000", token.INTEGER},
		{"0xDEAD_beef", token.INTEGER},
		{"0o755", token.INTEGER},
		{"0b1101_0110", token.INTEGER},
		{"01", token.ILLEGAL},
		{"1__000", token.ILLEGAL},
		{"1_", token.ILLEGAL},
		{"_1", token.ILLEGAL},
		{"+0x1", token.ILLEGAL},
		{"0x", token.ILLEGAL},
		{"0o8", token.ILLEGAL},
		{"0b2", token.ILLEGAL},
		{"0X1F", token.ILLEGAL},

		{"3.1415", token.FLOAT},
		{"-0.01", token.FLOAT},
		{"5e+22", token.FLOAT},
		{"1E06", token.FLOAT},
		{"-2e-2", token.FLOAT},
		{"6.626e-34", token.FLOAT},
		{"224_617.445_991", token.FLOAT},
		{"inf", token.FLOAT},
		{"+inf", token.FLOAT},
		{"-nan", token.FLOAT},
		{"1.", token.ILLEGAL},
		{".1", token.ILLEGAL},
		{"1.e5", token.ILLEGAL},
		{"1e", token.ILLEGAL},
		{"03.14", token.ILLEGAL},
		{"Inf", token.ILLEGAL},

		{"1979-05-27", token.LOCAL_DATE},
		{"07:32:00", token.LOCAL_TIME},
		{"00:32:00.999999", token.LOCAL_TIME},
		{"07:32", token.LOCAL_TIME}, // TOML 1.1: seconds optional
		{"1979-05-27T07:32:00", token.LOCAL_DATETIME},
		{"1979-05-27t07:32:00", token.LOCAL_DATETIME},
		{"1979-05-27 07:32:00", token.LOCAL_DATETIME},
		{"1979-05-27T07:32", token.LOCAL_DATETIME},
		{"1979-05-27T07:32:00Z", token.OFFSET_DATETIME},
		{"1979-05-27T07:32:00z", token.OFFSET_DATETIME},
		{"1979-05-27T00:32:00.999999-07:00", token.OFFSET_DATETIME},
		{"1979-05-27 07:32Z", token.OFFSET_DATETIME},
		{"1979-5-27", token.ILLEGAL},
		{"7:32:00", token.ILLEGAL},
		{"07:32:00.", token.ILLEGAL},
		{"1979-05-27T07:32:00+0700", token.ILLEGAL},
		{"1979-05-27T", token.ILLEGAL},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tok := valueToken(t, tt.input)
			require.Equal(t, tt.wantType, tok.Type)
			require.Equal(t, tt.input, tok.Literal)
		})
	}
}

func TestNextTokenStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType token.TokenType
	}{
		{"basic", `"hello"`, token.BASIC_STRING},
		{"basic empty", `""`, token.BASIC_STRING},
		{"basic escapes", `"\" \\ \b \f \n \r \t \u00E9 \U0001F600"`, token.BASIC_STRING},
		{"basic escape e", `"\e[0m"`, token.BASIC_STRING},    // TOML 1.1
		{"basic escape x", `"\x41\xff"`, token.BASIC_STRING}, // TOML 1.1
		{"basic unicode", `"héllo ✓"`, token.BASIC_STRING},
		{"basic tab", "\"a\tb\"", token.BASIC_STRING},
		{"basic unknown escape", `"\a"`, token.ILLEGAL},
		{"basic short x", `"\x4"`, token.ILLEGAL},
		{"basic short u", `"\u12"`, token.ILLEGAL},
		{"basic short U", `"\U0001F60"`, token.ILLEGAL},
		{"basic control char", "\"a\x01b\"", token.ILLEGAL},
		{"basic DEL", "\"a\x7fb\"", token.ILLEGAL},
		{"basic unterminated", `"abc`, token.ILLEGAL},
		{"basic escaped quote at end", `"abc\"`, token.ILLEGAL},

		{"literal", `'C:\Users\nodejs'`, token.LITERAL_STRING},
		{"literal empty", `''`, token.LITERAL_STRING},
		{"literal unterminated", `'abc`, token.ILLEGAL},
		{"literal control char", "'a\x00b'", token.ILLEGAL},

		{"ml basic", `"""hello"""`, token.ML_BASIC_STRING},
		{"ml basic newlines", "\"\"\"\nRoses\r\nViolets\"\"\"", token.ML_BASIC_STRING},
		{"ml basic inner quotes", `"""a "b" ""c"" d"""`, token.ML_BASIC_STRING},
		{"ml basic trailing quotes", `"""a"""""`, token.ML_BASIC_STRING},
		{"ml basic line-ending backslash", "\"\"\"a \\  \n   b\"\"\"", token.ML_BASIC_STRING},
		{"ml basic line-ending backslash CRLF", "\"\"\"a \\\r\n b\"\"\"", token.ML_BASIC_STRING},
		{"ml basic line-ending backslash lone CR", "\"\"\"x\\\n\rb\"\"\"", token.ILLEGAL},
		{"ml basic bad escape", `"""\q"""`, token.ILLEGAL},
		{"ml basic lone CR", "\"\"\"a\rb\"\"\"", token.ILLEGAL},
		{"ml basic unterminated", `"""abc""`, token.ILLEGAL},

		{"ml literal", `'''I [dw]on't need \d{2} apples'''`, token.ML_LITERAL_STRING},
		{"ml literal newlines", "'''\nfirst\nsecond'''", token.ML_LITERAL_STRING},
		{"ml literal trailing quotes", `'''a'''''`, token.ML_LITERAL_STRING},
		{"ml literal unterminated", `'''abc`, token.ILLEGAL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := valueToken(t, tt.input)
			require.Equal(t, tt.wantType, tok.Type)
			require.Equal(t, tt.input, tok.Literal)
		})
	}
}

func TestNextTokenQuotedKeys(t *testing.T) {
	input := `"a b".'c' = 1`
	requireTokens(t, input, []wantToken{
		{token.BASIC_STRING, `"a b"`},
		{token.DOT, "."},
		{token.LITERAL_STRING, `'c'`},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.INTEGER, "1"},
	})
}

func TestNextTokenTableHeaders(t *testing.T) {
	input := "[ server . 'alpha' ]\n[[products]]\n"
	requireTokens(t, input, []wantToken{
		{token.LBRACKET, "["},
		{token.WHITESPACE, " "},
		{token.BARE_KEY, "server"},
		{token.WHITESPACE, " "},
		{token.DOT, "."},
		{token.WHITESPACE, " "},
		{token.LITERAL_STRING, "'alpha'"},
		{token.WHITESPACE, " "},
		{token.RBRACKET, "]"},
		{token.NEWLINE, "\n"},
		{token.DOUBLE_LBRACKET, "[["},
		{token.BARE_KEY, "products"},
		{token.DOUBLE_RBRACKET, "]]"},
		{token.NEWLINE, "\n"},
	})
}

func TestNextTokenArrays(t *testing.T) {
	input := "a = [[1, 2], [true]]\nb = [\n  1979-05-27, # first\n  inf,\n]\nc = 1\n"
	requireTokens(t, input, []wantToken{
		{token.BARE_KEY, "a"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.LBRACKET, "["},
		{token.LBRACKET, "["},
		{token.INTEGER, "1"},
		{token.COMMA, ","},
		{token.WHITESPACE, " "},
		{token.INTEGER, "2"},
		{token.RBRACKET, "]"},
		{token.COMMA, ","},
		{token.WHITESPACE, " "},
		{token.LBRACKET, "["},
		{token.BOOL, "true"},
		{token.RBRACKET, "]"},
		{token.RBRACKET, "]"},
		{token.NEWLINE, "\n"},
		{token.BARE_KEY, "b"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.LBRACKET, "["},
		{token.NEWLINE, "\n"},
		{token.WHITESPACE, "  "},
		{token.LOCAL_DATE, "1979-05-27"},
		{token.COMMA, ","},
		{token.WHITESPACE, " "},
		{token.COMMENT, "# first"},
		{token.NEWLINE, "\n"},
		{token.WHITESPACE, "  "},
		{token.FLOAT, "inf"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.RBRACKET, "]"},
		{token.NEWLINE, "\n"},
		{token.BARE_KEY, "c"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.INTEGER, "1"},
		{token.NEWLINE, "\n"},
	})
}

func TestNextTokenInlineTables(t *testing.T) {
	// TOML 1.1: inline tables may span lines and have a trailing comma.
	input := "p = { true = true, x.y = [{ z = 1 }],\n  # note\n}\n"
	requireTokens(t, input, []wantToken{
		{token.BARE_KEY, "p"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.LBRACE, "{"},
		{token.WHITESPACE, " "},
		{token.BARE_KEY, "true"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.BOOL, "true"},
		{token.COMMA, ","},
		{token.WHITESPACE, " "},
		{token.BARE_KEY, "x"},
		{token.DOT, "."},
		{token.BARE_KEY, "y"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.LBRACKET, "["},
		{token.LBRACE, "{"},
		{token.WHITESPACE, " "},
		{token.BARE_KEY, "z"},
		{token.WHITESPACE, " "},
		{token.EQUALS, "="},
		{token.WHITESPACE, " "},
		{token.INTEGER, "1"},
		{token.WHITESPACE, " "},
		{token.RBRACE, "}"},
		{token.RBRACKET, "]"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.WHITESPACE, "  "},
		{token.COMMENT, "# note"},
		{token.NEWLINE, "\n"},
		{token.RBRACE, "}"},
		{token.NEWLINE, "\n"},
	})
}

func TestNextTokenComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []wantToken
	}{
		{"CRLF not part of comment", "# hi\r\n", []wantToken{
			{token.COMMENT, "# hi"},
			{token.NEWLINE, "\r\n"},
		}},
		{"unicode and tab", "#\tgrüße ✓", []wantToken{
			{token.COMMENT, "#\tgrüße ✓"},
		}},
		{"control char", "# a\x01b", []wantToken{
			{token.ILLEGAL, "# a\x01b"},
		}},
		{"DEL", "# a\x7f", []wantToken{
			{token.ILLEGAL, "# a\x7f"},
		}},
		{"invalid UTF-8", "# a\xff", []wantToken{
			{token.ILLEGAL, "# a\xff"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTokens(t, tt.input, tt.want)
		})
	}
}

func TestNextTokenIllegal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []wantToken
	}{
		{"unknown char", "a = @", []wantToken{
			{token.BARE_KEY, "a"},
			{token.WHITESPACE, " "},
			{token.EQUALS, "="},
			{token.WHITESPACE, " "},
			{token.ILLEGAL, "@"},
		}},
		{"lone CR", "a\rb", []wantToken{
			{token.BARE_KEY, "a"},
			{token.ILLEGAL, "\r"},
			{token.BARE_KEY, "b"},
		}},
		{"non-ASCII bare key", "ö = 1", []wantToken{
			{token.ILLEGAL, "\xc3"},
			{token.ILLEGAL, "\xb6"},
			{token.WHITESPACE, " "},
			{token.EQUALS, "="},
			{token.WHITESPACE, " "},
			{token.INTEGER, "1"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTokens(t, tt.input, tt.want)
		})
	}
}

const specExample = `# This is a TOML document

title = "TOML Example"

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00

[database]
enabled = true
ports = [ 8000, 8001, 8002 ]
data = [ ["delta", "phi"], [3.14] ]
temp_targets = { cpu = 79.5, case = 72.0 }

[servers]

[servers.alpha]
ip = "10.0.0.1"
role = """
frontend \
  proxy"""

[[products]]
name = 'Hammer'
sku = 738594937
`

// lexAll returns all tokens up to (excluding) EOF. It fails the test if the
// lexer stops making progress.
func lexAll(t *testing.T, input string) []token.Token {
	t.Helper()
	l := New(input)
	var toks []token.Token
	for range len(input) + 1 {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			return toks
		}
		require.NotEmpty(t, tok.Literal, "non-EOF token with empty literal at %d:%d", tok.Line, tok.Col)
		toks = append(toks, tok)
	}
	require.FailNow(t, "lexer did not reach EOF")
	return nil
}

func concatLiterals(toks []token.Token) string {
	var sb strings.Builder
	for _, tok := range toks {
		sb.WriteString(tok.Literal)
	}
	return sb.String()
}

func TestLexerIsLossless(t *testing.T) {
	toks := lexAll(t, specExample)
	for _, tok := range toks {
		require.NotEqual(t, token.TokenType(token.ILLEGAL), tok.Type, "unexpected ILLEGAL %q at %d:%d", tok.Literal, tok.Line, tok.Col)
	}
	require.Equal(t, specExample, concatLiterals(toks))
}

func FuzzLexer(f *testing.F) {
	f.Add(specExample)
	f.Add("a = [1, {b = 2}]\n[[c]]\n")
	f.Add("s = \"\"\"\\\n\"\"\"")
	f.Add("x = '''a''''' # c\r\n")
	f.Fuzz(func(t *testing.T, input string) {
		toks := lexAll(t, input)
		require.Equal(t, input, concatLiterals(toks))

		line, col := 1, 1
		for _, tok := range toks {
			require.Equal(t, line, tok.Line, "wrong line for %q", tok.Literal)
			require.Equal(t, col, tok.Col, "wrong col for %q", tok.Literal)
			for i := range len(tok.Literal) {
				col++
				if tok.Literal[i] == '\n' {
					line, col = line+1, 1
				}
			}
		}
	})
}
