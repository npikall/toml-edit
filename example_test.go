package tomledit_test

import (
	"errors"
	"fmt"

	tomledit "github.com/npikall/toml-edit"
)

const exampleConfig = `# server settings
[server]
host = "localhost"  # bind address
port = 8080
debug = true
`

func Example() {
	doc, err := tomledit.Parse(exampleConfig)
	if err != nil {
		panic(err)
	}
	port, _ := doc.GetInt("server", "port")
	fmt.Println("port:", port)

	_ = doc.Set([]string{"server", "port"}, 9090)
	_ = doc.Insert([]string{"server", "workers"}, 4)
	_ = doc.Delete("server", "debug")
	fmt.Print(doc)
	// Output:
	// port: 8080
	// # server settings
	// [server]
	// host = "localhost"  # bind address
	// port = 9090
	// workers = 4
}

func ExampleDocument_Get() {
	doc, _ := tomledit.Parse("tags = [\"a\", \"b\"]\npoint = { x = 1 }\n")
	tags, _ := doc.Get("tags")
	x, _ := doc.Get("point", "x")
	_, ok := doc.Get("missing")
	fmt.Println(tags, x, ok)
	// Output: [a b] 1 false
}

func ExampleDocument_Set() {
	doc, _ := tomledit.Parse("name = 'old'   # keep me\n")
	_ = doc.Set([]string{"name"}, "new")
	fmt.Print(doc)
	// Output: name = "new"   # keep me
}

func ExampleDocument_Insert() {
	doc, _ := tomledit.Parse("[a]\nx = 1\n")
	_ = doc.Insert([]string{"a", "point"}, map[string]any{"x": 1, "y": 2})
	_ = doc.Insert([]string{"b", "c", "d"}, []int{1, 2})
	err := doc.Insert([]string{"a", "x"}, 2)
	fmt.Print(doc)
	fmt.Println(errors.Is(err, tomledit.ErrExists))
	// Output:
	// [a]
	// x = 1
	// point = { x = 1, y = 2 }
	//
	// [b.c]
	// d = [1, 2]
	// true
}

func ExampleDocument_Delete() {
	doc, _ := tomledit.Parse(`point = { x = 1, y = 2 }

# old section
[legacy]
a = 1
`)
	_ = doc.Delete("point", "x")
	_ = doc.Delete("legacy")
	fmt.Print(doc)
	// Output: point = { y = 2 }
}

func ExampleStrict10() {
	_, err := tomledit.Parse("t = 07:32\n", tomledit.Strict10())
	fmt.Println(err)
	// Output: parse: 1:5: expected a value, found invalid "07:32"
}
