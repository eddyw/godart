// Copyright 2011 The Go Authors. All rights reserved.
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in ../overlay/std/LICENSE.

package driver_test

import (
	"encoding/json"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

type importerContext struct {
	CgoEnabled  bool     `json:"cgo_enabled"`
	BuildTags   []string `json:"build_tags"`
	ToolTags    []string `json:"tool_tags"`
	ReleaseTags []string `json:"release_tags"`
	Compiler    string   `json:"compiler"`
	GOOS        string   `json:"goos"`
	GOARCH      string   `json:"goarch"`
	GOROOT      string   `json:"goroot"`
}

type importerSource struct {
	Path          string `json:"path"`
	Source        string `json:"source"`
	SHA256        string `json:"sha256"`
	Adapted       string `json:"adapted"`
	AdaptedSHA256 string `json:"adapted_sha256"`
}

type importerInputs struct {
	Command     []string        `json:"command"`
	Output      string          `json:"output"`
	Context     importerContext `json:"context"`
	Source      importerSource  `json:"source"`
	TestSources load.Filter     `json:"-"`
}

const importerContextTemplate = `{{with context}}{{printf "{\"compiler\":%q,\"goos\":%q,\"goarch\":%q,\"goroot\":%q,\"cgo_enabled\":%t" .Compiler .GOOS .GOARCH .GOROOT .CgoEnabled}},"build_tags":[{{range $i, $tag := .BuildTags}}{{if $i}},{{end}}{{printf "%q" $tag}}{{end}}],"tool_tags":[{{range $i, $tag := .ToolTags}}{{if $i}},{{end}}{{printf "%q" $tag}}{{end}}],"release_tags":[{{range $i, $tag := .ReleaseTags}}{{if $i}},{{end}}{{printf "%q" $tag}}{{end}}]}{{end}}`

func prepareImporterInputs(t *testing.T, pkg string) *importerInputs {
	t.Helper()
	if pkg != "go/internal/gcimporter" && pkg != "go/internal/srcimporter" {
		return nil
	}
	inputs := &importerInputs{Command: []string{"go", "list", "-f=" + importerContextTemplate, "runtime"}}
	inputs.Output = runProgram(t, "", inputs.Command[0], inputs.Command[1:]...)
	if err := json.Unmarshal([]byte(inputs.Output), &inputs.Context); err != nil {
		t.Fatal(err)
	}
	ctx := inputs.Context
	if ctx.Compiler != "gc" || ctx.GOOS != build.Default.GOOS || ctx.GOARCH != build.Default.GOARCH || ctx.GOROOT != build.Default.GOROOT || ctx.CgoEnabled != build.Default.CgoEnabled || !slices.Equal(ctx.BuildTags, build.Default.BuildTags) || !slices.Equal(ctx.ToolTags, build.Default.ToolTags) || !slices.Equal(ctx.ReleaseTags, build.Default.ReleaseTags) {
		t.Fatalf("SDK context differs from the source inventory: %+v", ctx)
	}
	name := filepath.Base(pkg) + "_test.go"
	path := filepath.Join(build.Default.GOROOT, "src", filepath.FromSlash(pkg), name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var adapted string
	switch pkg {
	case "go/internal/gcimporter":
		adapted, err = importerCompilerSource(string(raw), ctx.Compiler)
	case "go/internal/srcimporter":
		adapted, err = importerArchitectureSource(string(raw), ctx)
	}
	if err != nil {
		t.Fatal(err)
	}
	inputs.Source = importerSource{Path: filepath.ToSlash(filepath.Join("src", pkg, name)), Source: string(raw), SHA256: languageDigest(raw), Adapted: adapted, AdaptedSHA256: languageDigest([]byte(adapted))}
	inputs.TestSources = load.Filter{path: []byte(adapted)}
	return inputs
}

const importerCompilerGuard = `	if runtime.Compiler != "gc" {
		t.Skipf("gc-built packages not available (compiler = %s)", runtime.Compiler)
	}`

func importerCompilerSource(source, compiler string) (string, error) {
	if !strings.Contains(source, importerCompilerGuard) {
		return "", fmt.Errorf("the SDK compiler availability guard changed")
	}
	adapted := strings.ReplaceAll(source, importerCompilerGuard, strings.ReplaceAll(importerCompilerGuard, "runtime.Compiler", strconv.Quote(compiler)))
	if strings.Contains(adapted, "runtime.") || strings.Count(adapted, "\n\t\"runtime\"\n") != 1 {
		return "", fmt.Errorf("runtime input appears outside the SDK availability guard")
	}
	return strings.Replace(adapted, "\n\t\"runtime\"\n", "\n", 1), nil
}

func importerArchitectureSource(source string, ctx importerContext) (string, error) {
	const declaration = "var importer = New(&build.Default, token.NewFileSet(), make(map[string]*types.Package))"
	if strings.Count(source, declaration) != 1 {
		return "", fmt.Errorf("the SDK source importer initializer changed")
	}
	const guard = "testenv.MustHaveCGO(t)"
	if strings.Count(source, guard) != 1 {
		return "", fmt.Errorf("the SDK cgo availability guard changed")
	}
	setup := "var _ = func() bool {\n\tbuild.Default.Compiler = " + strconv.Quote(ctx.Compiler) + "\n\tbuild.Default.GOARCH = " + strconv.Quote(ctx.GOARCH) + "\n\tbuild.Default.CgoEnabled = " + strconv.FormatBool(ctx.CgoEnabled) + "\n"
	for _, tags := range []struct {
		name   string
		values []string
	}{{"BuildTags", ctx.BuildTags}, {"ToolTags", ctx.ToolTags}, {"ReleaseTags", ctx.ReleaseTags}} {
		values := make([]string, len(tags.values))
		for i, value := range tags.values {
			values[i] = strconv.Quote(value)
		}
		setup += "\tbuild.Default." + tags.name + " = []string{" + strings.Join(values, ", ") + "}\n"
	}
	setup += "\treturn true\n}()\n\n"
	adapted := strings.Replace(source, declaration, setup+declaration, 1)
	return strings.Replace(adapted, guard, `if !build.Default.CgoEnabled { t.Skip("selected Go SDK has no cgo") }`, 1), nil
}

func TestImporterInputAdapters(t *testing.T) {
	source := "package p\nimport (\n\t\"runtime\"\n\t\"testing\"\n)\nfunc TestInput(t *testing.T) {\n" + importerCompilerGuard + "\n\tif value != 42 {t.Fatal(value)}\n}\n"
	adapted, err := importerCompilerSource(source, "gc")
	if err != nil || !strings.Contains(adapted, "if value != 42 {t.Fatal(value)}") || strings.Contains(adapted, "runtime.Compiler") {
		t.Fatalf("compiler input adapter: %v\n%s", err, adapted)
	}
	for _, changed := range []string{
		strings.Replace(source, "t.Skipf", "t.Fatalf", 1),
		source + "\nvar compiler = runtime.Compiler\n",
		strings.Replace(source, "runtime.Compiler !=", "runtime.Compiler ==", 1),
	} {
		if _, err := importerCompilerSource(changed, "gc"); err == nil {
			t.Fatal("accepted a compiler input outside the availability guard")
		}
	}
	const declaration = "var importer = New(&build.Default, token.NewFileSet(), make(map[string]*types.Package))"
	source = "func TestCgo(t *testing.T) { testenv.MustHaveCGO(t) }\n" + declaration
	adapted, err = importerArchitectureSource(source, importerContext{Compiler: "gc", GOARCH: "arm64", CgoEnabled: true, ToolTags: []string{"arm64.v8.0"}})
	if err != nil || !strings.HasSuffix(adapted, declaration) || !strings.Contains(adapted, `build.Default.GOARCH = "arm64"`) || !strings.Contains(adapted, `build.Default.CgoEnabled = true`) || !strings.Contains(adapted, `build.Default.ToolTags = []string{"arm64.v8.0"}`) {
		t.Fatalf("source input adapter: %v\n%s", err, adapted)
	}
	for _, changed := range []string{"", source + "\n" + declaration, declaration, strings.Replace(source, "MustHaveCGO", "MustHaveGoBuild", 1)} {
		if _, err := importerArchitectureSource(changed, importerContext{}); err == nil {
			t.Fatal("accepted a changed source importer initializer")
		}
	}
}
