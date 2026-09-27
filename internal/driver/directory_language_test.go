// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in internal/overlay/std/LICENSE.

package driver_test

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

type languageDirectoryPackage struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	ModulePath string   `json:"module_path,omitempty"`
	Files      []string `json:"files"`
}

func directoryModulePaths(packages []languageDirectoryPackage) map[string]string {
	remap := false
	for _, pkg := range packages {
		if strings.IndexFunc(pkg.Path, func(r rune) bool { return r > 127 }) >= 0 {
			remap = true
		}
	}
	paths := map[string]string{}
	for i := range packages {
		pkg := &packages[i]
		path := pkg.Path
		if remap {
			path = fmt.Sprintf("test/p%d", i)
			pkg.ModulePath = path
		}
		paths[pkg.Path] = path
	}
	return paths
}

func TestDirectoryModulePaths(t *testing.T) {
	for _, test := range []struct {
		name  string
		paths []string
		want  []string
	}{
		{"ASCII", []string{"test/a", "test/b"}, []string{"test/a", "test/b"}},
		{"Unicode", []string{"test/Þfoo", "test/Þmain"}, []string{"test/p0", "test/p1"}},
		{"collision", []string{"test/Þfoo", "test/p0"}, []string{"test/p0", "test/p1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			packages := make([]languageDirectoryPackage, len(test.paths))
			for i, path := range test.paths {
				packages[i].Path = path
			}
			paths := directoryModulePaths(packages)
			for i, pkg := range packages {
				if pkg.Path != test.paths[i] || paths[pkg.Path] != test.want[i] {
					t.Fatalf("package %d: original=%q mapped=%q", i, pkg.Path, paths[pkg.Path])
				}
				if test.name != "ASCII" && pkg.ModulePath != test.want[i] {
					t.Fatalf("package %d: missing module path evidence", i)
				}
			}
		})
	}
}

func TestDirectoryPackagesOfGoTest(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	var names []string
	for _, c := range inv.Cases {
		if c.Adapter == "directory-packages" {
			cases = append(cases, c)
			names = append(names, c.Name)
		}
	}
	selected := testShard(t, names)
	standard := run(t, "", "go", "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std")
	for _, action := range []string{"compiledir", "rundir", "errorcheckdir", "errorcheckandrundir"} {
		t.Run(action, func(t *testing.T) {
			diagnostics := action == "errorcheckdir" || action == "errorcheckandrundir"
			runPackages := action == "rundir" || action == "errorcheckandrundir"
			var matcher, matcherSource string
			if diagnostics {
				matcher, matcherSource = directoryDiagnosticMatcher(t)
			}
			var outcomes []*languageOutcome
			for _, c := range cases {
				if c.Action == action && selected[c.Name] {
					outcomes = append(outcomes, &languageOutcome{Fixture: c, Contract: action, Status: "unobserved"})
				}
			}
			retainLanguageEvidence(t, "vm", inv, outcomes)
			for _, result := range outcomes {
				t.Run(result.Fixture.Name, func(t *testing.T) {
					t.Parallel()
					t.Cleanup(func() {
						switch {
						case t.Failed():
							result.Status = "failed"
						case t.Skipped():
							result.Status = "excluded"
						case action == "errorcheckdir" && result.Fixture.ExpectedFailure:
							result.Status = "upstream-expected-failure"
						default:
							result.Status = "passed"
						}
					})
					c := result.Fixture
					if !c.NativeEligible || !c.GartEligible {
						t.Skipf("upstream build constraints: native=%t Gart=%t", c.NativeEligible, c.GartEligible)
					}
					root := filepath.Join(build.Default.GOROOT, "test")
					sourceDir := filepath.Join(root, c.Name+".dir")
					entries, err := os.ReadDir(sourceDir)
					if err != nil {
						t.Fatal(err)
					}
					grouped := map[string]int{}
					for _, entry := range entries {
						if filepath.Ext(entry.Name()) != ".go" {
							continue
						}
						f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(sourceDir, entry.Name()), nil, parser.PackageClauseOnly)
						if err != nil {
							t.Fatal(err)
						}
						index, ok := grouped[f.Name.Name]
						if c.Recipe == c.Action+" -s" || !ok {
							index = len(result.DirectoryPackages)
							grouped[f.Name.Name] = index
							result.DirectoryPackages = append(result.DirectoryPackages, languageDirectoryPackage{Name: f.Name.Name, Path: "test/" + strings.TrimSuffix(entry.Name(), ".go")})
						}
						result.DirectoryPackages[index].Files = append(result.DirectoryPackages[index].Files, entry.Name())
					}
					if len(result.DirectoryPackages) == 0 {
						t.Fatal("upstream directory has no Go packages")
					}
					nativeDir, dir, out := t.TempDir(), t.TempDir(), t.TempDir()
					if err := os.Mkdir(filepath.Join(nativeDir, "test"), 0755); err != nil {
						t.Fatal(err)
					}
					config := standard
					for _, pkg := range result.DirectoryPackages {
						config += "\npackagefile " + pkg.Path + "=" + filepath.Join(nativeDir, pkg.Path+".a")
					}
					cfg := filepath.Join(nativeDir, "importcfg")
					writeFile(t, cfg, config)
					result.ImportConfigSHA256 = languageDigest([]byte(config))
					rejectIndex := len(result.DirectoryPackages) - 1
					if action == "errorcheckandrundir" {
						rejectIndex--
					}
					for index, pkg := range result.DirectoryPackages {
						args := []string{"tool", "compile", "-e", "-D", "test", "-importcfg=" + cfg}
						if pkg.Name == "main" {
							args = append(args, "-p=main")
						} else {
							args = append(args, "-o", pkg.Path+".a", "-p", pkg.Path)
						}
						args = append(args, strings.Fields(inv.GCFlags)...)
						if diagnostics {
							args = append(args, "-d=panic")
						}
						if c.Recipe == "compiledir -c=2" || c.Recipe == "compiledir -c=4" || c.Recipe == "rundir -l=4" {
							args = append(args, strings.Fields(c.Recipe)[1])
						}
						for _, name := range pkg.Files {
							args = append(args, filepath.Join(sourceDir, name))
						}
						result.NativeCompile = append(result.NativeCompile, languageExecution{})
						execution := &result.NativeCompile[len(result.NativeCompile)-1]
						if diagnostics {
							result.NativeDiagnostics += recordLanguage(t, execution, nativeDir, runLimit, "go", args...)
							last := index == rejectIndex
							if execution.TimedOut || last && execution.Exit <= 0 || !last && execution.Exit != 0 {
								t.Fatalf("package %s: native exit %d, expected rejection=%t, timeout=%t", pkg.Path, execution.Exit, last, execution.TimedOut)
							}
						} else {
							runLanguage(t, execution, nativeDir, runLimit, "go", args...)
						}
					}
					if action == "errorcheckandrundir" {
						for index, first := range result.NativeCompile {
							result.NativeRecompile = append(result.NativeRecompile, languageExecution{})
							execution := &result.NativeRecompile[len(result.NativeRecompile)-1]
							recordLanguage(t, execution, nativeDir, runLimit, first.Command[0], first.Command[1:]...)
							if execution.TimedOut || index == rejectIndex && execution.Exit <= 0 || index != rejectIndex && execution.Exit != 0 {
								t.Fatalf("native repeat package %d: exit %d, timeout=%t", index, execution.Exit, execution.TimedOut)
							}
						}
					}
					if runPackages {
						last := result.DirectoryPackages[len(result.DirectoryPackages)-1]
						args := []string{"tool", "link", "-s", "-w", "-buildid=test", "-o", "native", "-importcfg=" + cfg}
						if c.Recipe == "rundir -ldflags -strictdups=2 -w=0" {
							args = append(args, "-strictdups=2", "-w=0")
						}
						args = append(args, strings.TrimSuffix(last.Files[0], ".go")+".o")
						runLanguage(t, &result.NativeLink, nativeDir, runLimit, "go", args...)
						want, err := os.ReadFile(filepath.Join(root, c.Name+".out"))
						if err != nil && !os.IsNotExist(err) {
							t.Fatal(err)
						}
						result.ExpectedSHA256 = languageDigest(want)
						got := runLanguage(t, &result.Native, nativeDir, runLimit, filepath.Join(nativeDir, "native"))
						if got != string(want) {
							t.Fatalf("native output differs from upstream fixture\nwant: %s\ngot: %s", want, got)
						}
					} else {
						result.Native = result.NativeCompile[len(result.NativeCompile)-1]
					}
					if c.Exclusion != nil {
						t.Skip(c.Exclusion.Classification + ": " + c.Exclusion.Reason)
					}

					result.ModuleSource = "module test\n\ngo " + strings.TrimPrefix(inv.GoVersion, "go") + "\n"
					result.ModuleSHA256 = languageDigest([]byte(result.ModuleSource))
					writeFile(t, filepath.Join(dir, "go.mod"), result.ModuleSource)
					license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
					if err != nil {
						t.Fatal(err)
					}
					writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
					result.AdaptedSources = map[string]string{}
					modulePaths := directoryModulePaths(result.DirectoryPackages)
					var patterns, imports []string
					for _, pkg := range result.DirectoryPackages {
						subdir := strings.TrimPrefix(modulePaths[pkg.Path], "test/")
						patterns = append(patterns, "./"+subdir)
						library := "package:test/" + subdir + ".dart"
						if pkg.Name == "main" {
							library = "bin/main.dart"
						}
						imports = append(imports, "import "+strconv.Quote(library)+";")
						for _, name := range pkg.Files {
							path := filepath.Join(sourceDir, name)
							data, err := os.ReadFile(path)
							if err != nil {
								t.Fatal(err)
							}
							fset := token.NewFileSet()
							f, err := parser.ParseFile(fset, path, data, parser.ImportsOnly)
							if err != nil {
								t.Fatal(err)
							}
							for i := len(f.Imports) - 1; i >= 0; i-- {
								lit := f.Imports[i].Path
								value, err := strconv.Unquote(lit.Value)
								if err != nil {
									t.Fatal(err)
								}
								if strings.HasPrefix(value, "./") {
									path := "test/" + strings.TrimPrefix(value, "./")
									if mapped, ok := modulePaths[path]; ok {
										path = mapped
									}
									replacement := strconv.Quote(path)
									start, end := fset.Position(lit.Pos()).Offset, fset.Position(lit.End()).Offset
									data = append(append(append([]byte{}, data[:start]...), replacement...), data[end:]...)
								}
							}
							targetName := name
							if strings.HasSuffix(name, "_test.go") {
								targetName += ".gart.go"
							}
							relative := filepath.Join(subdir, targetName)
							result.AdaptedSources[filepath.ToSlash(relative)] = string(data)
							target := filepath.Join(dir, relative)
							if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
								t.Fatal(err)
							}
							writeFile(t, target, string(data))
						}
					}
					start := time.Now()
					err = driver.Build(driver.Options{Dir: dir, Patterns: patterns, Out: out, Runtime: runtimeDir(t), Format: false})
					if diagnostics {
						result.Gart = languageExecution{Ran: true, Seconds: time.Since(start).Seconds()}
						if err != nil {
							result.Gart.Exit = 1
							result.BuildError = err.Error()
							text := err.Error()
							for _, pkg := range result.DirectoryPackages {
								subdir := strings.TrimPrefix(modulePaths[pkg.Path], "test/")
								for _, name := range pkg.Files {
									targetName := name
									if strings.HasSuffix(name, "_test.go") {
										targetName += ".gart.go"
									}
									text = strings.ReplaceAll(text, filepath.Join(dir, subdir, targetName), filepath.Join(sourceDir, name))
								}
							}
							result.GartDiagnostics = regexp.MustCompile(`(?m)^(.*?\.go:[0-9]+):[0-9]+:`).ReplaceAllString(text, "$1:")
						} else {
							t.Error("Gart accepted the invalid package")
						}
						result.Gart.OutputSHA256 = languageDigest([]byte(result.GartDiagnostics))
						result.WrapperSHA256 = languageDigest([]byte(matcherSource))
						matchDirectoryDiagnostics(t, result, matcher, sourceDir, dir)
						if action == "errorcheckdir" {
							return
						}
						rejected := result.Gart
						result.GartReject = &rejected
						result.Gart = languageExecution{}
						patterns = append(patterns[:rejectIndex], patterns[rejectIndex+1:]...)
						err = driver.Build(driver.Options{Dir: dir, Patterns: patterns, Out: out, Runtime: runtimeDir(t), Format: false})
					}
					if err != nil {
						result.BuildError = err.Error()
					}
					if want := c.Rejection; want != nil {
						if err == nil {
							t.Fatal("the build now passes: remove its expected rejection")
						}
						got := strings.Split(strings.ReplaceAll(err.Error(), dir, "$DIR"), "\n")
						slices.Sort(got)
						if !slices.Equal(got, want) {
							t.Fatalf("the rejection changed\nwant:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
						}
						t.Skipf("known rejection:\n%v", err)
					}
					if err != nil {
						t.Fatal(err)
					}
					prepareDart(t, out)
					if action == "compiledir" {
						wrapper := strings.Join(imports, "\n") + "\nvoid main() {}\n"
						result.WrapperSource = wrapper
						result.WrapperSHA256 = languageDigest([]byte(wrapper))
						writeFile(t, filepath.Join(out, "compile.dart"), wrapper)
						runLanguage(t, &result.Gart, out, runLimit, "dart", "compile", "kernel", "compile.dart", "-o", "compile.dill")
					} else {
						entry := filepath.Join(out, "bin", "main.dart")
						got := runLanguage(t, &result.Gart, nativeDir, runLimit, "dart", "--packages="+filepath.Join(out, ".dart_tool/package_config.json"), entry)
						if languageDigest([]byte(got)) != result.ExpectedSHA256 {
							t.Fatalf("Gart output differs from upstream fixture: %s", got)
						}
					}
				})
			}
		})
	}
}
