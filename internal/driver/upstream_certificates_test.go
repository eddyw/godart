package driver_test

import (
	"encoding/json"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

type certificateInput struct {
	Start      int    `json:"start"`
	End        int    `json:"end"`
	Expression string `json:"expression"`
	PEM        string `json:"pem"`
}

type certificateFixture struct {
	SourceSHA256  string `json:"source_sha256"`
	Patch         string `json:"patch"`
	AdaptedSource string `json:"adapted_source"`
	AdaptedSHA256 string `json:"adapted_sha256"`
}

type certificateInputs struct {
	TestSources     load.Filter         `json:"-"`
	RootFixture     *certificateFixture `json:"root_fixture,omitempty"`
	Source          string              `json:"source"`
	SourceSHA256    string              `json:"source_sha256"`
	GeneratorSHA256 string              `json:"generator_sha256"`
	GeneratorSource string              `json:"generator_source"`
	Generator       languageExecution   `json:"generator"`
	Output          string              `json:"output"`
	AdaptedSHA256   string              `json:"adapted_sha256"`
	Inputs          []certificateInput  `json:"inputs"`
	Overlay         string              `json:"-"`
}

func certificateCalls(t *testing.T, source string) []certificateInput {
	t.Helper()
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "verify_test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var inputs []certificateInput
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name.Name != "generatePEMCertWithRepeatSAN" {
			return true
		}
		start, end := fs.Position(call.Pos()).Offset, fs.Position(call.End()).Offset
		inputs = append(inputs, certificateInput{Start: start, End: end, Expression: source[start:end]})
		return true
	})
	return inputs
}

func applyCertificateInputs(t *testing.T, source string, inputs []certificateInput) string {
	t.Helper()
	calls := certificateCalls(t, source)
	if len(calls) != len(inputs) {
		t.Fatal("certificate setup calls changed")
	}
	for i := len(calls) - 1; i >= 0; i-- {
		call := calls[i]
		if call.Expression != inputs[i].Expression {
			t.Fatal("certificate setup arguments changed")
		}
		source = source[:call.Start] + strconv.Quote(inputs[i].PEM) + source[call.End:]
	}
	return source
}

func prepareCertificateInputs(t *testing.T, sourcePath string) *certificateInputs {
	t.Helper()
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &certificateInputs{Source: string(data), SourceSHA256: languageDigest(data)}
	fixture.Inputs = certificateCalls(t, fixture.Source)
	if len(fixture.Inputs) != 4 {
		t.Fatal("the original certificate setup changed")
	}
	generator := selectedUpstreamSource(t, sourcePath, []string{"generatePEMCertWithRepeatSAN"})
	generator = strings.Replace(generator, "package x509", "package main\nimport (\"encoding/json\"; \"os\"; . \"crypto/x509\")", 1)
	generator += "\nfunc main() { values := []string{\n"
	for _, input := range fixture.Inputs {
		generator += input.Expression + ",\n"
	}
	generator += "}; if err := json.NewEncoder(os.Stdout).Encode(values); err != nil { panic(err) } }\n"
	fixture.GeneratorSource = generator
	fixture.GeneratorSHA256 = languageDigest([]byte(generator))
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/certificateinputs\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), generator)
	license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	fixture.Output = runLanguage(t, &fixture.Generator, dir, runLimit, "go", "run", ".")
	var values []string
	if err := json.Unmarshal([]byte(fixture.Output), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != len(fixture.Inputs) {
		t.Fatal("incomplete certificate setup")
	}
	for i, value := range values {
		fixture.Inputs[i].PEM = value
	}
	adapted := applyCertificateInputs(t, fixture.Source, fixture.Inputs)
	fixture.TestSources = load.Filter{sourcePath: []byte(adapted)}
	fixture.AdaptedSHA256 = languageDigest([]byte(adapted))
	path := filepath.Join(dir, "verify_test.go")
	writeFile(t, path, adapted)
	overlay, err := json.Marshal(map[string]map[string]string{"Replace": {sourcePath: path}})
	if err != nil {
		t.Fatal(err)
	}
	fixture.Overlay = filepath.Join(dir, "overlay.json")
	writeFile(t, fixture.Overlay, string(overlay))
	return fixture
}

func prepareX509SuiteInputs(t *testing.T, dir string) *certificateInputs {
	t.Helper()
	inputs := prepareCertificateInputs(t, filepath.Join(dir, "verify_test.go"))
	if build.Default.GOOS != "linux" {
		return inputs
	}
	name := filepath.Join(dir, "root_linux_test.go")
	source, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := os.ReadFile("testdata/x509-root-linux.patch")
	if err != nil {
		t.Fatal(err)
	}
	adapted, err := load.ApplyPatch(source, patch, name)
	if err != nil {
		t.Fatal(err)
	}
	inputs.RootFixture = &certificateFixture{SourceSHA256: languageDigest(source), Patch: string(patch), AdaptedSource: string(adapted), AdaptedSHA256: languageDigest(adapted)}
	inputs.TestSources[name] = adapted
	path := filepath.Join(filepath.Dir(inputs.Overlay), "root_linux_test.go")
	writeFile(t, path, string(adapted))
	data, err := os.ReadFile(inputs.Overlay)
	if err != nil {
		t.Fatal(err)
	}
	var overlay struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &overlay); err != nil {
		t.Fatal(err)
	}
	overlay.Replace[name] = path
	data, err = json.Marshal(overlay)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, inputs.Overlay, string(data))
	return inputs
}
