// Command toml-test-decoder reads TOML on stdin and writes toml-test's tagged
// JSON on stdout. Invalid TOML is reported on stderr with exit code 1.
//
// The -toml flag selects the TOML version, "1.1" (default) or "1.0":
//
//	toml-test test -toml 1.0 -decoder "toml-test-decoder -toml 1.0"
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/internal/tagged"
	"github.com/npikall/toml-edit/parser"
)

var errVersion = errors.New(`unsupported TOML version, want "1.0" or "1.1"`)

func main() {
	version := flag.String("toml", "1.1", `TOML version: "1.0" or "1.1"`)
	flag.Parse()
	if err := run(*version, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(version string, in io.Reader, out io.Writer) error {
	var opts []parser.Option
	switch version {
	case "1.1":
	case "1.0":
		opts = append(opts, parser.Strict10())
	default:
		return fmt.Errorf("%w: %q", errVersion, version)
	}
	src, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	doc, err := parser.Parse(string(src), opts...)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	table, err := eval.Evaluate(doc)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}
	if err := json.NewEncoder(out).Encode(tagged.FromTable(table)); err != nil {
		return fmt.Errorf("write json: %w", err)
	}
	return nil
}
