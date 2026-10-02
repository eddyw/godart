package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNamedPanicTargets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/namedpanic\n\ngo 1.27\n")
	suffixes := []string{"Bool", "Complex128", "Complex64", "Float32", "Float64", "Int", "Int8", "Int16", "Int32", "Int64", "String", "Uint", "Uint8", "Uint16", "Uint32", "Uint64", "Uintptr"}
	var declarations []string
	for _, suffix := range suffixes {
		declarations = append(declarations, "My"+suffix, "panicCustom"+suffix)
	}
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/testdata/testprog/panicprint.go"), declarations)
	writeFile(t, filepath.Join(dir, "panic.go"), source)
	var main strings.Builder
	main.WriteString("package main\nimport \"os\"\nfunc main(){switch os.Getenv(\"PANIC_SCENARIO\"){\n")
	for _, suffix := range suffixes {
		main.WriteString("case \"" + suffix + "\":panicCustom" + suffix + "()\n")
	}
	main.WriteString(`case "BuiltinBool":panic(true)
case "BuiltinString":panic("Panic\nline two")
case "BuiltinFloat32":panic(float32(-93.7))
case "BuiltinComplex64":panic(complex64(0.11+3i))
case "BuiltinUint64":panic(uint64(1<<64-1))
case "Error":panic(panicError("error\nline two"))
case "Stringer":panic(panicStringer("string\nline two"))
case "ErrorStringer":panic(panicBoth("value"))
`)
	main.WriteString("}}\n")
	main.WriteString(`type panicError string
func(e panicError) Error() string{return string(e)}
type panicStringer string
func(e panicStringer) String() string{return string(e)}
type panicBoth string
func(e panicBoth) Error() string{return "error\nline two"}
func(e panicBoth) String() string{return "string\nline two"}
`)
	writeFile(t, filepath.Join(dir, "main.go"), main.String())
	suffixes = append(suffixes, "BuiltinBool", "BuiltinString", "BuiltinFloat32", "BuiltinComplex64", "BuiltinUint64", "Error", "Stringer", "ErrorStringer")
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	wants := make(map[string]programResult)
	for _, suffix := range suffixes {
		want := programOutcomeWithLimitEnv(t, dir, append(os.Environ(), "PANIC_SCENARIO="+suffix), runLimit, native)
		if want.Exit != 2 {
			t.Fatalf("Go panic %s: %+v", suffix, want)
		}
		wants[suffix] = want
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			for i, suffix := range suffixes {
				t.Run(suffix, func(t *testing.T) {
					env := append(os.Environ(), "PANIC_SCENARIO="+suffix)
					var result programResult
					if target == "vm" {
						result = upstreamOutcome(t, dir, env, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
					} else if i == 0 {
						result = runUpstreamTargetEnv(t, target, out, "bin/main.dart", dir, []string{}, env)
					} else {
						command, args := filepath.Join(out, "tests.exe"), []string{}
						if target == "js" {
							command, args = "node", []string{filepath.Join(out, "run.js")}
						}
						if target == "wasm" {
							command, args = "node", []string{filepath.Join(out, "run.mjs")}
						}
						result = upstreamOutcome(t, dir, env, upstreamLimit(), command, args...)
					}
					if result != wants[suffix] {
						t.Fatalf("named panic exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, wants[suffix].Text)
					}
				})
			}
		})
	}
}
