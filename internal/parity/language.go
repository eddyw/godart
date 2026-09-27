// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in internal/overlay/std/LICENSE.

package parity

import (
	"embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"go/version"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

//go:embed testdata/language-skips.json testdata/language-rejections.json
var languagePolicy embed.FS

type LanguagePolicy struct {
	Skips      map[string]Omission `json:"skips"`
	Rejections map[string][]string `json:"rejections"`
}

func ReadLanguagePolicy() (*LanguagePolicy, error) {
	policy := &LanguagePolicy{}
	for path, dest := range map[string]any{"language-skips.json": &policy.Skips, "language-rejections.json": &policy.Rejections} {
		data, err := languagePolicy.ReadFile("testdata/" + path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, dest); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

type LanguageCase struct {
	NativeFlags     []string     `json:"native_flags,omitempty"`
	RunFiles        []string     `json:"run_files,omitempty"`
	Arguments       []string     `json:"arguments,omitempty"`
	GeneratorFiles  []string     `json:"generator_files,omitempty"`
	ModuleGoVersion string       `json:"module_go_version,omitempty"`
	ExpectedFailure bool         `json:"upstream_expected_failure,omitempty"`
	HostTools       bool         `json:"host_tools,omitempty"`
	Companions      []string     `json:"companions,omitempty"`
	Exclusion       *Omission    `json:"exclusion,omitempty"`
	Rejection       []string     `json:"rejection,omitempty"`
	Name            string       `json:"name"`
	File            string       `json:"file"`
	Recipe          string       `json:"recipe"`
	Action          string       `json:"action"`
	NativeEligible  bool         `json:"native_eligible"`
	GartEligible    bool         `json:"gart_eligible"`
	Adapter         string       `json:"adapter"`
	Files           []SourceFile `json:"files"`
}

type LanguageInventory struct {
	GCFlags      string         `json:"go_gcflags"`
	GoVersion    string         `json:"go_version"`
	GOOS         string         `json:"goos"`
	GOARCH       string         `json:"goarch"`
	BuildTags    []string       `json:"build_tags"`
	ToolTags     []string       `json:"tool_tags"`
	Cgo          bool           `json:"cgo"`
	Runner       SourceFile     `json:"runner"`
	SourceSHA256 string         `json:"source_sha256"`
	Cases        []LanguageCase `json:"cases"`
}

// DiscoverLanguage inventories the recipes selected by the active Go test runner.
// An available adapter records no successful execution.
func DiscoverLanguage(ctx build.Context) (*LanguageInventory, error) {
	policy, err := ReadLanguagePolicy()
	if err != nil {
		return nil, err
	}
	const runner = "src/cmd/internal/testdir/testdir_test.go"
	source, err := os.ReadFile(filepath.Join(ctx.GOROOT, runner))
	if err != nil {
		return nil, err
	}
	dirs, err := languageDirectories(source)
	if err != nil {
		return nil, err
	}
	failures, err := languageExpectedFailures(source, ctx.GOARCH)
	if err != nil {
		return nil, err
	}
	version, err := os.ReadFile(filepath.Join(ctx.GOROOT, "VERSION"))
	if err != nil {
		return nil, err
	}
	inv := &LanguageInventory{GoVersion: strings.Split(string(version), "\n")[0], GCFlags: os.Getenv("GO_GCFLAGS"), GOOS: ctx.GOOS, GOARCH: ctx.GOARCH, BuildTags: slices.Clone(ctx.BuildTags), ToolTags: slices.Clone(ctx.ToolTags), Cgo: ctx.CgoEnabled, Runner: SourceFile{Path: runner, SHA256: digest(source)}}
	gart := ctx
	gart.BuildTags = append(slices.Clone(ctx.BuildTags), "dart")
	root := filepath.Join(ctx.GOROOT, "test")
	identity := runner + "\x00" + inv.Runner.SHA256 + "\n"
	for _, dir := range dirs {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			file := filepath.ToSlash(filepath.Join(dir, entry.Name()))
			data, err := os.ReadFile(filepath.Join(root, file))
			if err != nil {
				return nil, err
			}
			recipe := languageRecipe(string(data))
			fields := strings.Fields(recipe)
			if len(fields) == 0 {
				return nil, fmt.Errorf("missing upstream language recipe: %s", file)
			}
			native := languageEligible(string(data), ctx, inv.GCFlags)
			eligible, err := gart.MatchFile(filepath.Join(root, dir), entry.Name())
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			c := LanguageCase{ExpectedFailure: failures[file], Name: strings.TrimSuffix(file, ".go"), File: file, Recipe: recipe, Action: fields[0], NativeEligible: native, GartEligible: eligible, Adapter: "missing", Files: []SourceFile{{Path: file, SHA256: digest(data)}}}
			switch c.Name {
			case "fixedbugs/issue21317", "linkobj", "winbatch":
				c.HostTools = true
			case "linkmain_run":
				c.HostTools = true
				c.Companions = []string{"helloworld.go", "linkmain.go"}
			case "linkx_run":
				c.HostTools = true
				c.Companions = []string{"linkx.go"}
			}
			for _, companion := range c.Companions {
				data, err := os.ReadFile(filepath.Join(root, companion))
				if err != nil {
					return nil, err
				}
				c.Files = append(c.Files, SourceFile{Path: companion, SHA256: digest(data)})
			}
			if recipe == "run" {
				c.Adapter = "single-file"
			} else if flags := languageRunCompilerFlags[recipe]; flags != nil {
				c.Adapter = "run-recipe"
				c.NativeFlags = slices.Clone(flags)
			} else if files, args, ok := languageRunRecipe(recipe); ok {
				c.Adapter = "run-recipe"
				c.RunFiles, c.Arguments = files, args
				for _, name := range files {
					data, err := os.ReadFile(filepath.Join(root, name))
					if err != nil {
						return nil, err
					}
					c.Files = append(c.Files, SourceFile{Path: name, SHA256: digest(data)})
				}
			} else if recipe == "rundir" || recipe == "rundir -l=4" || recipe == "rundir -ldflags -strictdups=2 -w=0" || recipe == "compiledir" || recipe == "errorcheckdir" || recipe == "compiledir -s" || recipe == "compiledir -c=2" || recipe == "compiledir -c=4" || recipe == "errorcheckdir -s" {
				c.Adapter = "directory-packages"
			} else if slices.Contains([]string{
				"compile", "compile -p=p", "compile -p=main", "compile -N",
				"compile -B", "compile -c=2", "compile -c=4", "compile -l",
				"compile -d=ssa/check/on", "compile -N -d=softfloat",
				"compile -d=checkptr", "compile -d=libfuzzer",
				"compile -d=ssa/check/seed", "compile -d=ssa/check/seed=1",
				"compile -dynlink",
				"compile -godebug gotypesalias=1", "compile -goexperiment fieldtrack",
			}, recipe) {
				c.Adapter = "compile-file"
			} else if match := errorcheckVersionRecipe.FindStringSubmatch(recipe); match != nil {
				c.Adapter = "reject-file"
				c.ModuleGoVersion = strings.TrimPrefix(match[2], "go")
				if match[1] != "" {
					c.Adapter = "diagnostic-file"
				}
			} else if recipe == "buildrun" && c.Name == "fixedbugs/issue46234" {
				c.Adapter = "child-buildrun"
			} else if recipe == "build" || recipe == "build -gcflags=-l=4" || recipe == "build -goexperiment regabi,regabiargs" {
				c.Adapter = "build-file"
				if recipe == "build -gcflags=-l=4" {
					c.NativeFlags = []string{"-gcflags=-l=4"}
				}
			} else if recipe == "errorcheck" || recipe == "errorcheck -d=panic" || recipe == "errorcheck -e" || recipe == "errorcheck -e=0" || recipe == "errorcheck -t 10" || recipe == "errorcheck -complete" {
				c.Adapter = "reject-file"
			} else if files, ok := languageGeneratorFiles(recipe); ok {
				c.Adapter = "generated-file"
				if c.Action == "errorcheckoutput" {
					c.Adapter = "generated-rejection"
				}
				c.GeneratorFiles = files
				for _, name := range files {
					data, err := os.ReadFile(filepath.Join(root, name))
					if err != nil {
						return nil, err
					}
					c.Files = append(c.Files, SourceFile{Path: name, SHA256: digest(data)})
				}
			} else if moduleVersion, ok := languageModuleRecipe(recipe); ok {
				c.Adapter = "directory-module"
				c.ModuleGoVersion = moduleVersion
				if moduleVersion == "" {
					c.ModuleGoVersion, err = languageModuleVersion(source)
					if err != nil {
						return nil, err
					}
				}
			}
			for _, suffix := range []string{".dir", ".out"} {
				companion := filepath.Join(root, c.Name+suffix)
				if _, err := os.Stat(companion); os.IsNotExist(err) {
					continue
				} else if err != nil {
					return nil, err
				}
				err := filepath.WalkDir(companion, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						return nil
					}
					data, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					relative, err := filepath.Rel(root, path)
					if err != nil {
						return err
					}
					c.Files = append(c.Files, SourceFile{Path: filepath.ToSlash(relative), SHA256: digest(data)})
					return nil
				})
				if err != nil {
					return nil, err
				}
			}
			if c.Action == "asmcheck" {
				c.Exclusion = &Omission{Classification: "target-limit", Reason: "The assertions match native Go assembly instructions. Gart delegates machine code generation to Dart and does not preserve Go compiler instruction selection."}
			}
			if (c.Action == "builddir" || c.Action == "buildrundir") && slices.ContainsFunc(c.Files, func(file SourceFile) bool { return strings.HasSuffix(file.Path, ".s") }) {
				c.Exclusion = &Omission{Classification: "target-limit", Reason: "The recipe assembles and links native Go assembly files. Gart does not compile Go assembly or preserve the native linker ABI."}
			}
			switch c.Recipe {
			case "run -gcflags=-d=converthash=qy":
				c.Exclusion = &Omission{Classification: "target-limit", Reason: "The recipe selects experimental Go backend behavior for floating-point overflow conversions. The Go specification leaves these values implementation-dependent. Gart does not reproduce native compiler debug modes."}
			case "errorcheck -0 -m -l",
				"errorcheck -0 -m",
				"errorcheck -0 -m=2",
				"errorcheck -0 -N -m -l",
				"errorcheck -0 -l -m",
				"errorcheck -0 -m -m",
				"errorcheck -0 -m -l -smallframes",
				"errorcheck -0 -l -m=2":
				c.Exclusion = &Omission{Classification: "target-limit", Reason: "The assertions match native Go compiler optimization decisions. Gart delegates native optimization to Dart. Source compilation and runtime behavior require separate coverage."}
			case "errorcheck -+ -0 -l -d=wb",
				"errorcheck -0 -+ -p=internal/runtime/sys -m",
				"errorcheck -0 -N -d=nil",
				"errorcheck -0 -d=append,slice",
				"errorcheck -0 -d=checkptr -m",
				"errorcheck -0 -d=escapedebug=1",
				"errorcheck -0 -d=nil",
				"errorcheck -0 -d=ssa/check_bce/debug=1",
				"errorcheck -0 -d=ssa/intrinsics/debug",
				"errorcheck -0 -d=ssa/known_bits/debug=1",
				"errorcheck -0 -d=ssa/late_fuse/debug=1",
				"errorcheck -0 -d=ssa/opt/debug=1",
				"errorcheck -0 -d=ssa/phiopt/debug=3",
				"errorcheck -0 -d=ssa/prove/debug=1",
				"errorcheck -0 -d=ssa/prove/debug=2",
				"errorcheck -0 -d=ssa/tighten/debug=1",
				"errorcheck -0 -d=tailcall=1",
				"errorcheck -0 -d=typeassert",
				"errorcheck -0 -d=wb",
				"errorcheck -0 -l -d=defer",
				"errorcheck -0 -l -d=wb",
				"errorcheck -0 -l -m -live",
				"errorcheck -0 -live",
				"errorcheck -0 -live -l",
				"errorcheck -0 -live -wb=0",
				"errorcheck -0 -m -d=escapemutationscalls,zerocopy -l",
				"errorcheck -0 -m -live -std",
				"errorcheckdir -0 -m",
				"errorcheckwithauto -0 -l -live -wb=0 -d=ssa/insert_resched_checks/off",
				"errorcheckwithauto -0 -m -d=inlfuncswithclosures=1":
				c.Exclusion = &Omission{Classification: "target-limit", Reason: "The assertions inspect native Go optimization decisions or runtime metadata. Gart delegates native optimization and runtime metadata to Dart. Source compilation and runtime behavior require separate coverage."}
			}
			if skip, ok := policy.Skips[c.Name]; ok {
				c.Exclusion = &skip
			}
			if c.Name == "fixedbugs/issue15609" && (ctx.GOARCH == "amd64" || ctx.GOARCH == "386") {
				c.Exclusion = &Omission{Classification: "target-limit", Reason: "The selected jump function requires native x86 assembly. Other architectures select an original Go body."}
			}
			c.Rejection = policy.Rejections[c.Name]
			inv.Cases = append(inv.Cases, c)
		}
	}
	slices.SortFunc(inv.Cases, func(a, b LanguageCase) int { return strings.Compare(a.Name, b.Name) })
	for _, c := range inv.Cases {
		for _, f := range c.Files {
			identity += f.Path + "\x00" + f.SHA256 + "\n"
		}
	}
	inv.SourceSHA256 = digest([]byte(identity))
	return inv, nil
}

var errorcheckVersionRecipe = regexp.MustCompile(`^errorcheck(\s+-0)?\s+-lang=(go[0-9]+\.[0-9]+)$`)

var directoryRecipe = regexp.MustCompile(`^runindir(?:\s+-gomodversion\s+(?:"([^"]*)"|'([^']*)'|([^\s"']+)))?$`)

var languageRunCompilerFlags = map[string][]string{
	"run -gcflags -l=4":              {"-gcflags", "-l=4"},
	`run -gcflags="-l"`:              {"-gcflags=-l"},
	"run -gcflags='all=-N -l'":       {"-gcflags=all=-N -l"},
	"run -gcflags=-d=converthash=qy": {"-gcflags=-d=converthash=qy"},
	"run -gcflags=-d=softfloat":      {"-gcflags=-d=softfloat"},
	"run -gcflags=-d=ssa/check/on":   {"-gcflags=-d=ssa/check/on"},
	"run -gcflags=-l=4":              {"-gcflags=-l=4"},
}

func languageRunRecipe(recipe string) ([]string, []string, bool) {
	fields := strings.Fields(recipe)
	if len(fields) < 2 || fields[0] != "run" {
		return nil, nil, false
	}
	var files, args []string
	for _, value := range fields[1:] {
		if strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\\\"'") {
			return nil, nil, false
		}
		if len(args) == 0 && strings.HasSuffix(value, ".go") {
			if !filepath.IsLocal(value) || slices.Contains(strings.Split(value, "/"), "..") {
				return nil, nil, false
			}
			files = append(files, filepath.ToSlash(filepath.Clean(value)))
		} else {
			args = append(args, value)
		}
	}
	return files, args, true
}

func languageGeneratorFiles(recipe string) ([]string, bool) {
	fields := strings.Fields(recipe)
	if len(fields) == 0 || (fields[0] != "runoutput" && fields[0] != "errorcheckoutput") {
		return nil, false
	}
	var files []string
	for _, name := range fields[1:] {
		if !filepath.IsLocal(name) || slices.Contains(strings.Split(name, "/"), "..") || strings.ContainsAny(name, "\\\"'") || strings.HasPrefix(name, "-") || !strings.HasSuffix(name, ".go") {
			return nil, false
		}
		files = append(files, filepath.ToSlash(filepath.Clean(name)))
	}
	return files, true
}

func languageModuleRecipe(recipe string) (string, bool) {
	parts := directoryRecipe.FindStringSubmatch(recipe)
	if parts == nil {
		return "", false
	}
	for _, part := range parts[1:] {
		if part != "" {
			return part, version.IsValid("go" + part)
		}
	}
	return "", true
}

func languageModuleVersion(source []byte) (string, error) {
	tree, err := parser.ParseFile(token.NewFileSet(), "testdir_test.go", source, 0)
	if err != nil {
		return "", err
	}
	var moduleVersion string
	ast.Inspect(tree, func(n ast.Node) bool {
		clause, ok := n.(*ast.CaseClause)
		if !ok || len(clause.List) != 1 {
			return true
		}
		label, ok := clause.List[0].(*ast.BasicLit)
		if !ok || label.Value != `"runindir"` {
			return true
		}
		for _, statement := range clause.Body {
			fallback, ok := statement.(*ast.IfStmt)
			if !ok || len(fallback.Body.List) != 1 {
				continue
			}
			condition, ok := fallback.Cond.(*ast.BinaryExpr)
			if !ok || condition.Op != token.EQL {
				continue
			}
			name, ok := condition.X.(*ast.Ident)
			if !ok || name.Name != "modVersion" {
				continue
			}
			empty, ok := condition.Y.(*ast.BasicLit)
			if !ok || empty.Value != `""` {
				continue
			}
			assignment, ok := fallback.Body.List[0].(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
				continue
			}
			name, ok = assignment.Lhs[0].(*ast.Ident)
			if !ok || name.Name != "modVersion" {
				continue
			}
			literal, ok := assignment.Rhs[0].(*ast.BasicLit)
			if ok && literal.Kind == token.STRING {
				moduleVersion, _ = strconv.Unquote(literal.Value)
			}
		}
		return false
	})
	if !version.IsValid("go" + moduleVersion) {
		return "", fmt.Errorf("upstream runindir recipe has no literal module version")
	}
	return moduleVersion, nil
}

func languageRecipe(source string) string {
	for line := range strings.SplitSeq(source, "\n") {
		if constraint.IsGoBuild(line) || constraint.IsPlusBuild(line) {
			continue
		}
		if recipe := strings.TrimSpace(strings.TrimPrefix(line, "//")); recipe != "" {
			return recipe
		}
	}
	return ""
}

func languageDirectories(source []byte) ([]string, error) {
	tree, err := parser.ParseFile(token.NewFileSet(), "testdir_test.go", source, 0)
	if err != nil {
		return nil, err
	}
	for _, decl := range tree.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != "dirs" || len(value.Values) != 1 {
				continue
			}
			list, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				break
			}
			var dirs []string
			for _, item := range list.Elts {
				literal, ok := item.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return nil, fmt.Errorf("nonliteral upstream test directory")
				}
				dir, err := strconv.Unquote(literal.Value)
				if err != nil {
					return nil, err
				}
				dirs = append(dirs, dir)
			}
			return dirs, nil
		}
	}
	return nil, fmt.Errorf("upstream language runner has no literal directory list")
}

// The upstream testdir runner ignores filename suffixes and evaluates all
// header constraints with its own tags, including test_run.
func languageEligible(source string, ctx build.Context, gcflags string) bool {
	match := func(tag string) bool {
		return tag == ctx.GOOS || tag == ctx.GOARCH || tag == "gc" || tag == "test_run" ||
			tag == "cgo" && ctx.CgoEnabled ||
			tag == "gcflags_noopt" && (strings.Contains(gcflags, "-N") || strings.Contains(gcflags, "-l")) ||
			slices.Contains(ctx.ReleaseTags, tag) || strings.HasPrefix(tag, "goexperiment.") && slices.Contains(ctx.ToolTags, tag)
	}
	for line := range strings.SplitSeq(source, "\n") {
		if strings.HasPrefix(line, "package ") {
			break
		}
		if expr, err := constraint.Parse(line); err == nil && !expr.Eval(match) {
			return false
		}
	}
	return true
}

func languageExpectedFailures(source []byte, arch string) (map[string]bool, error) {
	tree, err := parser.ParseFile(token.NewFileSet(), "testdir_test.go", source, 0)
	if err != nil {
		return nil, err
	}
	names := []string{"types2Failures"}
	switch arch {
	case "386", "arm", "mips", "mipsle":
		names = append(names, "types2Failures32Bit")
	}
	failures := map[string]bool{}
	for _, name := range names {
		found := false
		for _, declaration := range tree.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.VAR {
				continue
			}
			for _, item := range group.Specs {
				value, ok := item.(*ast.ValueSpec)
				if !ok || len(value.Names) != 1 || value.Names[0].Name != name || len(value.Values) != 1 {
					continue
				}
				call, ok := value.Values[0].(*ast.CallExpr)
				if !ok {
					return nil, fmt.Errorf("upstream failure list %s changed shape", name)
				}
				fn, ok := call.Fun.(*ast.Ident)
				if !ok || fn.Name != "setOf" {
					return nil, fmt.Errorf("upstream failure list %s no longer uses setOf", name)
				}
				for _, argument := range call.Args {
					literal, ok := argument.(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						return nil, fmt.Errorf("nonliteral upstream failure in %s", name)
					}
					path, err := strconv.Unquote(literal.Value)
					if err != nil {
						return nil, err
					}
					failures[path] = true
				}
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("upstream language runner has no %s list", name)
		}
	}
	return failures, nil
}
