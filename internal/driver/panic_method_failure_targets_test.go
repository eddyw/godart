package driver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestPanicMethodFailureTargets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/panicmethod\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "os"
type E string
func(E) Error() string { return fail() }
type S string
func(S) String() string { return fail() }
type B string
func(B) Error() string { return fail() }
func(B) String() string { return "must not use String" }
type X string
func fail() string {
 switch os.Getenv("PANIC_FAILURE") {
 case "int":panic(7)
 case "named":panic(X("print failed"))
 case "nil":panic(nil)
 case "pointer":var p *int;panic(p)
 case "newline":panic("print failed\nline two")
 }
 panic("print failed")
}
type orderedError string
func(e orderedError) Error() string { println("error invoked"); return string(e) }
type orderedStringer string
func(s orderedStringer) String() string {
 println("stringer invoked")
 if os.Getenv("PANIC_FAILURE") != "valid" { panic(nil) }
 return string(s)
}
func main(){switch os.Getenv("PANIC_METHOD"){case "chain":defer func(){panic(orderedStringer("outer"))}();panic(orderedError("inner"));case "error":panic(E("original"));case "both":panic(B("original"));default:panic(S("original"))}}
`)
	cases := []struct{ name, method, failure, panicnil string }{
		{"ErrorString", "error", "string", "0"},
		{"ErrorInt", "error", "int", "0"},
		{"ErrorNamedString", "error", "named", "0"},
		{"ErrorNilDefault", "error", "nil", "0"},
		{"ErrorNilLegacy", "error", "nil", "1"},
		{"StringerString", "stringer", "string", "0"},
		{"StringerInt", "stringer", "int", "0"},
		{"StringerNamedString", "stringer", "named", "0"},
		{"StringerNilDefault", "stringer", "nil", "0"},
		{"StringerNilLegacy", "stringer", "nil", "1"},
		{"ErrorBeforeStringer", "both", "string", "0"},
		{"ErrorPointer", "error", "pointer", "0"},
		{"ErrorNewline", "error", "newline", "0"},
		{"ChainValid", "chain", "valid", "0"},
		{"ChainNilDefault", "chain", "nil", "0"},
		{"ChainNilLegacy", "chain", "nil", "1"},
	}
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	wants := map[string]programResult{}
	for _, c := range cases {
		env := append(os.Environ(), "PANIC_METHOD="+c.method, "PANIC_FAILURE="+c.failure, "GODEBUG=panicnil="+c.panicnil)
		result := programOutcomeWithLimitEnv(t, dir, env, runLimit, native)
		if result.Exit != 2 {
			t.Fatalf("Go %s: %+v", c.name, result)
		}
		wants[c.name] = result
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			for i, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					env := append(os.Environ(), "PANIC_METHOD="+c.method, "PANIC_FAILURE="+c.failure, "GODEBUG=panicnil="+c.panicnil)
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
					if result != wants[c.name] {
						t.Fatalf("method panic exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, wants[c.name].Text)
					}
				})
			}
		})
	}
}
