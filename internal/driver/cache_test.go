package driver_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// TestCache proves the 2 hashes of stage 6. An equal source hash skips the
// compile, and an equal export hash keeps the cache of the dependents.
func TestCache(t *testing.T) {
	source := copyModule(t, "../../testdata/programs/multi")
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	build := func() *driver.Stats {
		t.Helper()
		stats, err := driver.BuildWithStats(driver.Options{
			Dir:      source,
			Patterns: []string{"./..."},
			Out:      out,
			Runtime:  runtime,
			Format:   false,
		})
		if err != nil {
			t.Fatal(err)
		}
		return stats
	}

	first := build()
	if len(first.Compiled) != 3 || len(first.Cached) != 0 {
		t.Fatalf("the first build gave %v compiled and %v cached", first.Compiled, first.Cached)
	}

	second := build()
	if len(second.Compiled) != 0 || len(second.Cached) != 3 {
		t.Fatalf("the second build gave %v compiled and %v cached", second.Compiled, second.Cached)
	}

	// A change inside a function body keeps the exported interface equal, thus
	// the dependents keep their cache.
	write(t, filepath.Join(source, "count", "count.go"), func(text string) string {
		return strings.Replace(text, "n := 0", "n := 0\n\t_ = n", 1)
	})
	third := build()
	if !slices.Equal(third.Compiled, []string{"example.com/multi/count"}) {
		t.Errorf("a change of a body compiled %v, want only the package itself", third.Compiled)
	}

	// A change of the exported interface reaches each dependent.
	write(t, filepath.Join(source, "count", "count.go"), func(text string) string {
		return strings.Replace(text, "func Runes(text string) int {", "func Runes(text string, extra int) int {", 1)
	})
	write(t, filepath.Join(source, "main.go"), func(text string) string {
		return strings.Replace(text, `count.Runes("héllo")`, `count.Runes("héllo", 0)`, 1)
	})
	fourth := build()
	if !slices.Contains(fourth.Compiled, "example.com/multi") ||
		!slices.Contains(fourth.Compiled, "example.com/multi/count") {
		t.Errorf("a change of the interface compiled %v, want the package and its dependent", fourth.Compiled)
	}
	if slices.Contains(fourth.Compiled, "example.com/multi/greet") {
		t.Errorf("the package greet compiled again, want the cache")
	}
}

// TestCacheBlockDigest proves that a package compiles again when the block
// analysis of the module changes.
//
// That analysis reads the whole module. A new function of a package that this
// package does not import can give a function value of a shared signature the
// async form, and each call through such a value then changes.
func TestCacheBlockDigest(t *testing.T) {
	source := t.TempDir()
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "lib"), 0o777); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(source, "go.mod"), "module example.com/blockcache\n\ngo 1.27\n")
	writeFile(t, filepath.Join(source, "lib", "lib.go"),
		"package lib\n\nfunc Apply(f func(int) bool, v int) bool { return f(v) }\n")
	writeFile(t, filepath.Join(source, "main.go"),
		"package main\n\nimport \"example.com/blockcache/lib\"\n\n"+
			"func main() { println(lib.Apply(func(i int) bool { return i > 0 }, 1)) }\n")

	out := t.TempDir()
	build := func() *driver.Stats {
		t.Helper()
		stats, err := driver.BuildWithStats(driver.Options{
			Dir:      source,
			Patterns: []string{"./..."},
			Out:      out,
			Runtime:  runtime,
			Format:   false,
		})
		if err != nil {
			t.Fatal(err)
		}
		return stats
	}

	build()
	second := build()
	if len(second.Compiled) != 0 {
		t.Fatalf("the second build compiled %v, want the cache", second.Compiled)
	}

	// The new function reaches a value of the signature that Apply calls, thus
	// Apply takes the async form.
	writeFile(t, filepath.Join(source, "main.go"),
		"package main\n\nimport \"example.com/blockcache/lib\"\n\n"+
			"var ch = make(chan int, 1)\n\n"+
			"func wait(i int) bool { <-ch; return i > 0 }\n\n"+
			"func main() {\n"+
			"\tprintln(lib.Apply(func(i int) bool { return i > 0 }, 1))\n"+
			"\tch <- 1\n"+
			"\tvar g func(int) bool = wait\n"+
			"\tprintln(g(2))\n"+
			"}\n")
	third := build()
	if !slices.Contains(third.Compiled, "example.com/blockcache/lib") {
		t.Errorf("the third build compiled %v, want the package lib", third.Compiled)
	}
}

func copyModule(t *testing.T, source string) string {
	t.Helper()
	out := t.TempDir()
	root, err := filepath.Abs(source)
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		target := filepath.Join(out, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o777)
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, text, 0o666)
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func write(t *testing.T, path string, change func(string) string) {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(change(string(text))), 0o666); err != nil {
		t.Fatal(err)
	}
}

// TestTestCache proves that a test build reads the cache and gives byte equal
// output. The identifier of a test variant holds the import path of the package
// under test, thus the cache and the output path read the identifier.
func TestTestCache(t *testing.T) {
	source, err := filepath.Abs("../../testdata/tests/basic")
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	options := driver.Options{
		Dir:      source,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtime,
		Format:   true,
		Tests:    true,
	}
	first, err := driver.BuildWithStats(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Compiled) == 0 || len(first.Cached) != 0 {
		t.Fatalf("the first build gave %v compiled and %v cached", first.Compiled, first.Cached)
	}
	for _, want := range []string{
		"example.com/basic/calc",
		"example.com/basic/calc [example.com/basic/calc.test]",
		"example.com/basic/calc_test [example.com/basic/calc.test]",
		"example.com/basic/calc.test",
	} {
		if !slices.Contains(first.Compiled, want) {
			t.Errorf("the build compiled %v, want the package %q", first.Compiled, want)
		}
	}
	if len(first.Binaries) != 1 || first.Binaries[0].Entry != filepath.Join("bin", "calc_test.dart") {
		t.Errorf("got the binaries %v, want 1 entry point bin/calc_test.dart", first.Binaries)
	}
	before := tree(t, out)
	second, err := driver.BuildWithStats(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Compiled) != 0 {
		t.Errorf("the second build compiled %v, want the cache", second.Compiled)
	}
	if got := tree(t, out); got != before {
		t.Error("the 2 builds gave a different output")
	}
}

// tree reads each Dart file of a directory and joins the content. Equal inputs
// must give byte equal output.
func tree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".dart") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b.WriteString(rel)
		b.WriteString("\x00")
		b.Write(text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
