package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimeCPUTargets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/runtimecpu\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"runtime")
func main(){
 runtime.GOMAXPROCS(1)
 count:=runtime.NumCPU()
 fmt.Println(count, runtime.GOMAXPROCS(0))
 runtime.Gosched()
 fmt.Println(runtime.NumCPU()==count)
 runtime.GOMAXPROCS(1)
 fmt.Println(runtime.NumCPU(), runtime.GOMAXPROCS(-1))
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	nativeWant := programOutcome(t, dir, native)
	wasm := filepath.Join(t.TempDir(), "main.wasm")
	build := programOutcomeWithLimitEnv(t, dir, append(os.Environ(), "GOOS=js", "GOARCH=wasm"), runLimit, "go", "build", "-o", wasm, ".")
	if build.Exit != 0 {
		t.Fatalf("Go js/wasm build failed: %+v", build)
	}
	// Go js/wasm cannot start with GOMAXPROCS greater than 1.
	webWant := programOutcomeWithLimitEnv(t, dir, append(os.Environ(), "GOMAXPROCS=1"), runLimit, filepath.Join(runtime.GOROOT(), "lib", "wasm", "go_js_wasm_exec"), wasm)
	if nativeWant.Exit != 0 || webWant.Exit != 0 {
		t.Fatalf("Go CPU oracle failed: native=%+v web=%+v", nativeWant, webWant)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			want := nativeWant
			if target == "js" || target == "wasm" {
				want = webWant
			}
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("CPU query exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, want.Text)
			}
		})
	}
}
