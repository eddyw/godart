package load

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
	// Dir is the directory of the package under test. A test binary runs in
	// that directory, exactly as go test does, thus a relative path of a test
	// file reads the same file.
	Dir string
	// File is the path that the go command gives to its own entry point. The
	// overlay replaces the content of that path.
	File string
	// Source is the Go source of the entry point of the compiler.
	Source []byte
	// Tests lists the test functions in the order of the run.
	Tests []string
	// Benchmarks lists the benchmark functions in the order of the run.
	Benchmarks []string
	// FuzzTargets lists the fuzz targets in the order of the run.
	FuzzTargets []string
	FuzzCorpus  map[string][]fuzzInput
	// Cover holds the instrumented source of each file of the package under
	// test. A key is the file path that the go command reports. The map is
	// empty for a load that asked for no coverage.
	Cover map[string][]byte
	// CoverVars names the counter variable of each file, in the order of the
	// files. The entry point reads them.
	CoverVars []coverVar
	// CoverMode names the mode of the coverage, or it is empty for a load that
	// asked for none. A package with no statement still holds the mode, thus
	// the run reports the count of Go.
	CoverMode string
}

// coverVar names the counter variable of 1 file of a package under test.
type coverVar struct {
	// Name is the exported name of the variable that go tool cover wrote.
	Name string
	// File is the path of the source file that the report names.
	File string
}

// testBinaries builds the entry point of each test binary of the load.
//
// A load that asked for coverage also writes a counter into each statement of
// each package under test. Read coverPackage.
func testBinaries(found []*packages.Package, cover bool, files map[string][]byte, environment []string) ([]*Binary, error) {
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
		bin := &Binary{
			ID:    id,
			Under: under,
			Dir:   packageDir(byID[under+" ["+p.PkgPath+"]"], byID[under], byID[under+"_test ["+p.PkgPath+"]"]),
			File:  p.CompiledGoFiles[0],
		}
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
		bin.FuzzTargets = append(append([]string{}, inside.fuzzTargets...), outside.fuzzTargets...)
		bin.FuzzCorpus, err = readTestCorpus(bin.Dir, bin.FuzzTargets)
		if err != nil {
			return nil, err
		}
		if cover {
			bin.CoverMode = "set"
			if err := coverPackage(bin, files, environment, byID[under+" ["+p.PkgPath+"]"], byID[under]); err != nil {
				return nil, err
			}
		}
		bin.Source = testMainSource(inside, outside, bin)
		out = append(out, bin)
	}
	return out, nil
}

// packageDir gives the directory of the first package that holds a Go file.
func packageDir(pkgs ...*packages.Package) string {
	for _, p := range pkgs {
		if p == nil {
			continue
		}
		for _, name := range p.GoFiles {
			return filepath.Dir(name)
		}
	}
	return ""
}

// functions holds the functions of 1 test variant that the entry point reads.
type functions struct {
	// pkgPath is empty when the variant does not exist.
	pkgPath     string
	tests       []string
	benchmarks  []string
	fuzzTargets []string
	examples    []example
	// testMain marks a TestMain function of the variant. Such a function runs
	// the tests, thus the entry point calls it.
	testMain bool
}

// example names 1 example function with an output comment. Go runs such an
// example and compares the output with the text of the comment.
type example struct {
	name      string
	output    string
	unordered bool
}

// any reports whether the variant holds a function that the entry point names.
func (f *functions) any() bool {
	return len(f.tests) > 0 || len(f.benchmarks) > 0 ||
		len(f.fuzzTargets) > 0 || len(f.examples) > 0 || f.testMain
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
				out.fuzzTargets = append(out.fuzzTargets, name)
			}
		}
	}
	// Go runs an example that carries an output comment, and it compiles each
	// other 1 without a run.
	for _, ex := range doc.Examples(files...) {
		if ex.Output != "" || ex.EmptyOutput {
			out.examples = append(out.examples, example{
				name:      "Example" + ex.Name,
				output:    ex.Output,
				unordered: ex.Unordered,
			})
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
// imports only the package testing, the package regexp, and the 2 test
// variants. Read the decisions of the workstream doc.
//
// The entry point gives the harness the match function of regexp. The package
// testing imports no regexp, because regexp/syntax imports testing in its own
// test and the 2 imports would give a cycle.
func testMainSource(inside, outside *functions, bin *Binary) []byte {
	// A run with coverage reads the counter of each file of the package under
	// test, thus the entry point imports that package also when each test
	// stands in the external variant.
	coverPath := inside.pkgPath
	if coverPath == "" {
		coverPath = bin.Under
	}
	counters := len(bin.CoverVars) > 0 && coverPath != ""
	cover := bin.CoverMode != ""
	holdsInside := inside.any() || counters
	var b bytes.Buffer
	b.WriteString("// Code generated by gart. Do not edit.\n\n")
	b.WriteString("package main\n\n")
	b.WriteString("import (\n\t\"testing\"\n\n\t_ \"testing/internal/testdeps\"\n")
	if holdsInside {
		fmt.Fprintf(&b, "\n\tgartTest %q\n", coverPath)
	}
	if outside.any() {
		if !holdsInside {
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
	if len(bin.FuzzTargets) > 0 {
		writeFuzzCorpus(&b, bin.FuzzTargets, bin.FuzzCorpus)
	}
	if cover {
		b.WriteString("\tgartSetCover()\n")
	}
	fmt.Fprintf(&b, "\t%s\n", call)
	b.WriteString("\t\t[]testing.InternalTest{\n")
	writeList(&b, "gartTest", "F", inside.tests)
	writeList(&b, "gartXtest", "F", outside.tests)
	b.WriteString("\t\t},\n")
	b.WriteString("\t\t[]testing.InternalBenchmark{\n")
	writeList(&b, "gartTest", "F", inside.benchmarks)
	writeList(&b, "gartXtest", "F", outside.benchmarks)
	b.WriteString("\t\t},\n")
	b.WriteString("\t\t[]testing.InternalFuzzTarget{\n")
	writeList(&b, "gartTest", "Fn", inside.fuzzTargets)
	writeList(&b, "gartXtest", "Fn", outside.fuzzTargets)
	b.WriteString("\t\t},\n")
	b.WriteString("\t\t[]testing.InternalExample{\n")
	writeExamples(&b, "gartTest", inside.examples)
	writeExamples(&b, "gartXtest", outside.examples)
	b.WriteString("\t\t},\n")
	b.WriteString(tail)
	b.WriteString("}\n")
	if cover {
		b.WriteString(coverSource(bin, "gartTest", counters))
	}
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

// writeList writes 1 element for each function of a list. The field names the
// member that holds the function, and InternalFuzzTarget calls it Fn.
func writeList(b *bytes.Buffer, prefix, field string, names []string) {
	for _, name := range names {
		fmt.Fprintf(b, "\t\t\t{Name: %q, %s: %s.%s},\n", name, field, prefix, name)
	}
}

func writeExamples(b *bytes.Buffer, prefix string, list []example) {
	for _, ex := range list {
		fmt.Fprintf(b,
			"\t\t\t{Name: %q, F: %s.%s, Output: %q, Unordered: %v},\n",
			ex.name, prefix, ex.name, ex.output, ex.unordered)
	}
}

type fuzzInput struct{ name, data string }

func readTestCorpus(dir string, targets []string) (map[string][]fuzzInput, error) {
	corpus := map[string][]fuzzInput{}
	for _, target := range targets {
		path := filepath.Join(dir, "testdata", "fuzz", target)
		entries, err := os.ReadDir(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(path, entry.Name()))
			if err != nil {
				return nil, err
			}
			corpus[target] = append(corpus[target], fuzzInput{entry.Name(), string(data)})
		}
	}
	return corpus, nil
}

func writeFuzzCorpus(b *bytes.Buffer, names []string, corpus map[string][]fuzzInput) {
	b.WriteString("\ttesting.FuzzCorpus = map[string][]testing.InternalFuzzInput{\n")
	for _, name := range names {
		fmt.Fprintf(b, "\t\t%q: {", name)
		for _, input := range corpus[name] {
			fmt.Fprintf(b, "{Name: %q, Data: %q},", input.name, input.data)
		}
		b.WriteString("},\n")
	}
	b.WriteString("\t}\n")
}
