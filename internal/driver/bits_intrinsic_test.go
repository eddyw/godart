package driver_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestBitsIntrinsicCalls(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(runtime.GOROOT(), "src/math/bits/bits.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	bounds := map[string][2]int{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && (function.Name.Name == "Add64" || function.Name.Name == "Mul64") {
			bounds[function.Name.Name] = [2]int{fset.Position(function.Pos()).Line, fset.Position(function.End()).Line}
		}
	}
	if len(bounds) != 2 {
		t.Fatalf("SDK arithmetic declarations: %v", bounds)
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/bitsintrinsics\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), fmt.Sprintf(`package main

import (
	"fmt"
	"math/bits"
	"path/filepath"
	"reflect"
	"runtime"
)

var order int

const addStart, addEnd, mulStart, mulEnd = %d, %d, %d, %d

func operand(tag int, value uint64) uint64 {
	order = order*10 + tag
	_, _, line, _ := runtime.Caller(1)
	fmt.Println("operand", tag, line)
	return value
}

func operands() (uint64, uint64, uint64) {
	order = order*10 + 4
	return ^uint64(0), 1, 1
}

func factors() (uint64, uint64) {
	order = order*10 + 5
	return ^uint64(0), ^uint64(0)
}

func main() {
	sum, carry := bits.Add64(
		operand(1, ^uint64(0)),
		operand(2, 1),
		operand(3, 1),
	)
	fmt.Println("direct", sum, carry, order)
	sum, carry = bits.Add64(operands())
	fmt.Println("spread", sum, carry, order)
	hi, lo := bits.Mul64(factors())
	fmt.Println("product", hi, lo, order)
	word, out := bits.Add(^uint(0), 1, 1)
	fmt.Println("word", word, out)
	high, low := bits.Mul(^uint(0), ^uint(0))
	fmt.Println("word product", high, low)
	small, smallCarry := bits.Add32(^uint32(0), 1, 1)
	fmt.Println("small", small, smallCarry)
	for _, function := range []func(uint64, uint64, uint64) (uint64, uint64){bits.Add64} {
		a, b := function(^uint64(0), 1, 1)
		pc := reflect.ValueOf(function).Pointer()
		info := runtime.FuncForPC(pc)
		file, line := info.FileLine(pc)
		fmt.Println("function", a, b, info.Name(), filepath.Base(file), line, line >= addStart && line <= addEnd)
	}
	for _, function := range []func(uint64, uint64) (uint64, uint64){bits.Mul64} {
		a, b := function(^uint64(0), ^uint64(0))
		pc := reflect.ValueOf(function).Pointer()
		info := runtime.FuncForPC(pc)
		file, line := info.FileLine(pc)
		fmt.Println("function", a, b, info.Name(), filepath.Base(file), line, line >= mulStart && line <= mulEnd)
	}
}
`, bounds["Add64"][0], bounds["Add64"][1], bounds["Mul64"][0], bounds["Mul64"][1]))
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := programOutcome(t, dir, native)
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	metadata := regexp.MustCompile(`(?m)^function .* math/bits\.(?:Add64|Mul64) bits\.go \d+ true$`)
	if len(metadata.FindAllString(want.Text, -1)) != 2 {
		t.Fatalf("native arithmetic source metadata:\n%s", want.Text)
	}
	t.Logf("native instruction metadata:\n%s", want.Text)
	// Synthetic PCs identify source declarations, without native offsets.
	want.Text = metadata.ReplaceAllStringFunc(want.Text, func(line string) string {
		fields := strings.Fields(line)
		declaration := bounds[strings.TrimPrefix(fields[3], "math/bits.")][0]
		return fmt.Sprintf("%s %d true", strings.Join(fields[:5], " "), declaration)
	})
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("arithmetic calls exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, want.Text)
			}
		})
	}
}
