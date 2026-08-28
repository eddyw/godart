package load

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

// TestMainName is the stable name of the generated entry point of a test
// binary. The go command puts its own entry point in the build cache, and that
// path holds a hash. A stable name keeps the output equal on each machine.
const TestMainName = "gart_testmain.go"

// Binary is the entry point of the test binary of 1 package.
type Binary struct {
	// ID is the identifier of the main package, for example `m/sub.test`.
	ID string
	// Under is the import path of the package under test.
	Under string
	// File is the path that the go command gives to its own entry point. The
	// overlay replaces the content of that path.
	File string
	// Source is the Go source of the entry point of the compiler.
	Source []byte
	// Tests lists the test functions in the order of the run.
	Tests []string
	// Benchmarks lists the benchmark functions in the order of the run.
	Benchmarks []string
	// Skips lists each function that the binary reports as a skipped test.
	Skips []string
}

// testBinaries builds the entry point of each test binary of the load.
func testBinaries(found []*packages.Package) ([]*Binary, error) {
	byID := map[string]*packages.Package{}
	packages.Visit(found, nil, func(p *packages.Package) { byID[p.ID] = p })
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []*Binary
	for _, id := range ids {
		p := byID[id]
		if p.Name != "main" || !strings.HasSuffix(p.PkgPath, ".test") {
			continue
		}
		if len(p.CompiledGoFiles) != 1 {
			return nil, fmt.Errorf("the test binary %s holds %d files, and 1 is the count",
				p.PkgPath, len(p.CompiledGoFiles))
		}
		under := strings.TrimSuffix(p.PkgPath, ".test")
		bin := &Binary{ID: id, Under: under, File: p.CompiledGoFiles[0]}
		inside, err := functionsOf(byID[under+" ["+p.PkgPath+"]"])
		if err != nil {
			return nil, err
		}
		outside, err := functionsOf(byID[under+"_test ["+p.PkgPath+"]"])
		if err != nil {
			return nil, err
		}
		bin.Tests = append(append([]string{}, inside.tests...), outside.tests...)
		bin.Benchmarks = append(append([]string{}, inside.benchmarks...), outside.benchmarks...)
		bin.Skips = append(append([]string{}, inside.skips...), outside.skips...)
		bin.Source = testMainSource(inside, outside)
		out = append(out, bin)
	}
	return out, nil
}

// functions holds the functions of 1 test variant that the entry point reads.
type functions struct {
	// pkgPath is empty when the variant does not exist.
	pkgPath    string
	tests      []string
	benchmarks []string
	skips      []string
	// testMain marks a TestMain function of the variant. Such a function runs
	// the tests, thus the entry point calls it.
	testMain bool
}

// any reports whether the variant holds a function that the entry point names.
func (f *functions) any() bool {
	return len(f.tests) > 0 || len(f.benchmarks) > 0 || f.testMain
}

// functionsOf reads the test functions of 1 test variant.
func functionsOf(p *packages.Package) (*functions, error) {
	out := &functions{}
	if p == nil {
		return out, nil
	}
	out.pkgPath = p.PkgPath
	fset := token.NewFileSet()
	names := make([]string, 0, len(p.CompiledGoFiles))
	for _, name := range p.CompiledGoFiles {
		if strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var files []*ast.File
	for _, name := range names {
		file, err := parser.ParseFile(fset, name, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	for _, file := range files {
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Type.TypeParams != nil {
				continue
			}
			name := fn.Name.Name
			switch {
			case name == "TestMain" && params(fn) == 1:
				out.testMain = true
			case isTestName(name, "Test") && params(fn) == 1:
				out.tests = append(out.tests, name)
			case isTestName(name, "Benchmark") && params(fn) == 1:
				out.benchmarks = append(out.benchmarks, name)
			case isTestName(name, "Fuzz") && params(fn) == 1:
				out.skips = append(out.skips, name)
			}
		}
	}
	for _, ex := range doc.Examples(files...) {
		if ex.Output != "" || ex.EmptyOutput {
			out.skips = append(out.skips, "Example"+ex.Name)
		}
	}
	return out, nil
}

func params(fn *ast.FuncDecl) int {
	if fn.Type.Params == nil {
		return 0
	}
	count := 0
	for _, f := range fn.Type.Params.List {
		count += max(len(f.Names), 1)
	}
	return count
}

// isTestName reports whether a function name names a test of this kind. Go
// asks for a name that starts with the prefix, and the rune after the prefix
// must not be a lower case letter.
func isTestName(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	rest := name[len(prefix):]
	if rest == "" {
		return true
	}
	return !unicode.IsLower([]rune(rest)[0])
}

// testMainSource writes the Go source of the entry point of 1 test binary.
//
// The go command fixes the import list of its own entry point, thus the source
// imports only the package testing and the 2 test variants. Read the decisions
// of the workstream doc.
func testMainSource(inside, outside *functions) []byte {
	var b bytes.Buffer
	b.WriteString("// Code generated by gart. Do not edit.\n\n")
	b.WriteString("package main\n\n")
	b.WriteString("import (\n\t\"testing\"\n")
	if inside.any() {
		fmt.Fprintf(&b, "\n\tgartTest %q\n", inside.pkgPath)
	}
	if outside.any() {
		if !inside.any() {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "\tgartXtest %q\n", outside.pkgPath)
	}
	b.WriteString(")\n\n")
	call := "testing.Main("
	tail := "\t)\n"
	if prefix := testMainPrefix(inside, outside); prefix != "" {
		call = "testing.MainWith("
		tail = fmt.Sprintf("\t\t%s.TestMain,\n\t)\n", prefix)
	}
	b.WriteString("func main() {\n")
	fmt.Fprintf(&b, "\t%s\n", call)
	b.WriteString("\t\t[]testing.InternalTest{\n")
	writeList(&b, "gartTest", inside.tests)
	writeList(&b, "gartXtest", outside.tests)
	b.WriteString("\t\t},\n")
	b.WriteString("\t\t[]testing.InternalBenchmark{\n")
	writeList(&b, "gartTest", inside.benchmarks)
	writeList(&b, "gartXtest", outside.benchmarks)
	b.WriteString("\t\t},\n")
	b.WriteString(tail)
	b.WriteString("}\n")
	return b.Bytes()
}

// testMainPrefix names the import prefix of the variant that holds the TestMain
// function. The result is empty when no variant holds one.
//
// Go rejects a TestMain function in both variants of 1 package.
func testMainPrefix(inside, outside *functions) string {
	if inside.testMain {
		return "gartTest"
	}
	if outside.testMain {
		return "gartXtest"
	}
	return ""
}

func writeList(b *bytes.Buffer, prefix string, names []string) {
	for _, name := range names {
		fmt.Fprintf(b, "\t\t\t{Name: %q, F: %s.%s},\n", name, prefix, name)
	}
}

// SkipCause tells why gart does not run a function of a test file.
func SkipCause(name string) string {
	if strings.HasPrefix(name, "Fuzz") {
		return "gart runs no fuzz target"
	}
	return "gart runs no example"
}
