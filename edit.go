// Package tomledit reads and edits TOML documents while preserving their
// formatting: comments, whitespace, key quoting and the spelling of values
// that are not changed survive every edit byte for byte.
package tomledit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/format"
	"github.com/npikall/toml-edit/parser"
)

var (
	// ErrNotFound is returned when a key path does not exist.
	ErrNotFound = errors.New("key not found")
	// ErrNotValue is returned by Set when a key path names a table defined
	// by a header or dotted keys, or an array of tables.
	ErrNotValue = errors.New("key does not hold a value")
	// ErrType is returned by the typed getters when the value has a
	// different type.
	ErrType = errors.New("wrong value type")
)

// Document is a parsed TOML document. Its String method returns the source
// with all edits applied.
type Document struct {
	cst  *ast.Document
	root *eval.Table
}

// Parse parses and validates src. The result's String method returns src
// unchanged until it is edited.
func Parse(src string) (*Document, error) {
	cst, err := parser.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	root, err := eval.Evaluate(cst)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}
	return &Document{cst: cst, root: root}, nil
}

// String returns the document as TOML text.
func (d *Document) String() string { return d.cst.String() }

// Get returns the value at the key path: a string, int64, float64, bool,
// time.Time, eval.LocalDateTime, eval.LocalDate, eval.LocalTime, []any or
// *eval.Table. No keys return the root table.
func (d *Document) Get(keys ...string) (any, bool) {
	var v any = d.root
	for _, key := range keys {
		t, ok := v.(*eval.Table)
		if !ok {
			return nil, false
		}
		if v, ok = t.Get(key); !ok {
			return nil, false
		}
	}
	return v, true
}

// GetInt returns the integer at the key path.
func (d *Document) GetInt(keys ...string) (int64, error) { return get[int64](d, keys) }

// GetFloat returns the float at the key path.
func (d *Document) GetFloat(keys ...string) (float64, error) { return get[float64](d, keys) }

// GetString returns the string at the key path.
func (d *Document) GetString(keys ...string) (string, error) { return get[string](d, keys) }

// GetBool returns the boolean at the key path.
func (d *Document) GetBool(keys ...string) (bool, error) { return get[bool](d, keys) }

// Set replaces the value of an existing key. Only the value's text changes;
// the key, the whitespace around the value and a trailing comment are kept.
func (d *Document) Set(path []string, value any) error {
	if err := d.set(path, value); err != nil {
		return fmt.Errorf("%s: %w", dotted(path), err)
	}
	return nil
}

func (d *Document) set(path []string, value any) error {
	kv, err := d.keyValue(path)
	if err != nil {
		return err
	}
	node, err := format.Value(value)
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}
	*ast.ValueDecor(node) = *ast.ValueDecor(kv.Value)
	old := kv.Value
	kv.Value = node
	root, err := eval.Evaluate(d.cst)
	if err != nil {
		kv.Value = old
		return fmt.Errorf("evaluate: %w", err)
	}
	d.root = root
	return nil
}

// keyValue returns the CST node that defines the value at path.
func (d *Document) keyValue(path []string) (*ast.KeyValue, error) {
	if len(path) == 0 {
		return nil, ErrNotFound
	}
	parent, _ := d.Get(path[:len(path)-1]...)
	t, ok := parent.(*eval.Table)
	if !ok {
		return nil, ErrNotFound
	}
	last := path[len(path)-1]
	if _, ok := t.Get(last); !ok {
		return nil, ErrNotFound
	}
	kv, ok := t.KeyValue(last)
	if !ok {
		return nil, ErrNotValue
	}
	return kv, nil
}

// get returns the value at keys if it has type T.
//
//nolint:ireturn // T is a concrete value type chosen by the typed getters.
func get[T any](d *Document, keys []string) (T, error) {
	var zero T
	v, ok := d.Get(keys...)
	if !ok {
		return zero, fmt.Errorf("%s: %w", dotted(keys), ErrNotFound)
	}
	typed, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("%s: %w: got %T, want %T", dotted(keys), ErrType, v, zero)
	}
	return typed, nil
}

// dotted renders a key path for error messages.
func dotted(path []string) string { return strings.Join(path, ".") }
