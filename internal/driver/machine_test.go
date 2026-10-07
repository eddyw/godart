package driver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestMachineArrayInitialization(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module arrayframe\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "fmt"
type row struct { n int }
type table struct { rows [32]row }
func update(p *table, n int) {
again:
	p.rows[n] = row{n + 1}
	if n > 0 { n--; goto again }
}
func suspended(ch chan int) (result [3]int) {
	var zero [3]int
	<-ch
	var p table
	update(&p, 2)
	result[0] = p.rows[2].n
	result[1] = zero[1]
	return
}
func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	fmt.Println(suspended(ch))
	var p table
	update(&p, 2)
	fmt.Println(p.rows[0].n, p.rows[1].n, p.rows[2].n, p.rows[3].n)
}
`)
	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t),
	}); err != nil {
		t.Fatal(err)
	}
	found := false
	err := filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".part.dart") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		found = found || strings.Contains(string(data), "late List<row$>")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("the machine eagerly initializes its array temporaries")
	}
	run(t, out, "dart", "pub", "get")
	want := runProgram(t, dir, "go", "run", ".")
	if got := runProgram(t, out, "dart", "bin/main.dart"); got != want {
		t.Fatalf("Go:\n%s\nDart:\n%s", want, got)
	}
}
