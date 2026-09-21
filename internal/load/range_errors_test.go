package load_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestRangeClauseDiagnostics(t *testing.T) {
	for _, tt := range []struct{ name, source string }{
		{"slice", "func f(x []int) { for i,j,k := range x { _,_=i,j } }"},
		{"channel", "func f(x chan int) { for i,j,k := range x { _,_=i,j } }"},
		{"generic_slice", "func f[T ~[]int](x T) { for i,j,k := range x { _,_=i,j } }"},
		{"generic_channel", "func f[T ~chan int](x T) { for i,j,k := range x { _,_=i,j } }"},
		{"integer", "func f() { for i,j,k := range 5 { _,_=i,j } }"},
		{"iterator_zero", "func f(x func(func() bool)) { for i,j,k := range x { _,_=i,j } }"},
		{"iterator_one", "func f(x func(func(int) bool)) { for i,j,k := range x { _,_=i,j } }"},
		{"iterator_two", "func f(x func(func(int,int) bool)) { for i,j,k := range x { _,_=i,j } }"},
		{"invalid_range", "func f() { for i,j,k := range true { _,_=i,j } }"},
		{"undefined", "func f() { for i,j,k := range missing { _,_=i,j } }"},
		{"multiline", "func f(x []int) {\n for i,j,\n k,l := range x { _,_=i,j }\n for a,b,c := range x { _,_=a,b; _=missing }\n}\nvar after = undefined"},
		{"line_directive", "//line input.go:50\nfunc f(x []int) { for i,j,k := range x { _,_=i,j } }"},
		{"comments", "func f(x []int) { for i,j, /* extra */ k := range x { _,_=i,j } }"},
		{"compound", "func f(x []int) { var a [4]int; for a[0],a[1],a[2],a[3] = range x {} }"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/rangeerrors\n\ngo 1.27\n"), 0600); err != nil {
				t.Fatal(err)
			}
			name := filepath.Join(dir, "input.go")
			if err := os.WriteFile(name, []byte("package p\n"+tt.source+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "tool", "compile", "-e", "-o", filepath.Join(dir, "input.o"), name)
			cmd.Dir = dir
			want, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("native Go accepted an invalid range clause")
			}
			_, err = load.Load(dir, ".")
			if err == nil {
				t.Fatal("Gart accepted an invalid range clause")
			}
			normalize := func(s string) string {
				lines := strings.Split(strings.TrimSpace(s), "\n")
				sort.Strings(lines)
				return strings.Join(lines, "\n")
			}
			if got := normalize(err.Error()); got != normalize(string(want)) {
				t.Fatalf("range diagnostics differ:\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
