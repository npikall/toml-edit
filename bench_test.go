package tomledit_test

import (
	"fmt"
	"strings"
	"testing"

	tomledit "github.com/npikall/toml-edit"
)

// benchDoc returns a config with n table sections, each mixing comments,
// scalars, an array and an inline table.
func benchDoc(n int) string {
	var sb strings.Builder
	sb.WriteString("# generated benchmark config\ntitle = \"bench\"\n")
	for i := range n {
		fmt.Fprintf(&sb, `
# service %[1]d
[service%[1]d]
name = "svc-%[1]d" # display name
port = %[2]d
ratio = 0.75
enabled = true
started = 1979-05-27T07:32:00Z
tags = [
  "a", # first
  "b",
]
limits = { cpu = 2, memory = "1Gi" }
`, i, 8000+i)
	}
	return sb.String()
}

func benchParse(b *testing.B, src string) *tomledit.Document {
	b.Helper()
	doc, err := tomledit.Parse(src)
	if err != nil {
		b.Fatal(err)
	}
	return doc
}

func BenchmarkParse(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		src := benchDoc(n)
		b.Run(fmt.Sprintf("tables=%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			for b.Loop() {
				benchParse(b, src)
			}
		})
	}
}

func BenchmarkString(b *testing.B) {
	src := benchDoc(100)
	doc := benchParse(b, src)
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		_ = doc.String()
	}
}

func BenchmarkGet(b *testing.B) {
	doc := benchParse(b, benchDoc(100))
	for b.Loop() {
		if _, err := doc.GetInt("service50", "port"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSet(b *testing.B) {
	doc := benchParse(b, benchDoc(100))
	path := []string{"service50", "port"}
	for i := 0; b.Loop(); i++ {
		if err := doc.Set(path, i); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertDelete(b *testing.B) {
	doc := benchParse(b, benchDoc(100))
	path := []string{"service50", "extra"}
	for b.Loop() {
		if err := doc.Insert(path, "x"); err != nil {
			b.Fatal(err)
		}
		if err := doc.Delete(path...); err != nil {
			b.Fatal(err)
		}
	}
}
