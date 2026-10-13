package driver_test

import (
	"fmt"
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

func TestSyscallExit(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("process exit requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/exit\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"strconv";"syscall")
func stop(code int){defer fmt.Println("bad nested defer");syscall.Exit(code);fmt.Println("bad continuation")}
func main(){
 code,err:=strconv.ParseInt(os.Getenv("EXIT_CODE"),10,64);if err!=nil{panic(err)}
 fmt.Println("before exit",code);defer fmt.Println("bad main defer")
 switch os.Getenv("EXIT_MODE"){
 case "direct":stop(int(code))
 case "recover":func(){defer func(){fmt.Println("bad recover");recover()}();stop(int(code))}()
 case "defer":defer syscall.Exit(int(code));return
 case "goroutine":go stop(int(code));select{}
 default:panic("invalid mode")
 }
 fmt.Println("bad return")
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	cases := []struct {
		mode string
		code int64
	}{{"direct", 0}, {"direct", 7}, {"direct", -1}, {"direct", 255}, {"direct", 256}, {"direct", 257}, {"direct", 1<<32 | 7}, {"direct", 1 << 31}, {"direct", 1<<63 - 1}, {"recover", 7}, {"defer", 7}, {"goroutine", 7}}
	wants := make([]programResult, len(cases))
	for i, tc := range cases {
		env := append(os.Environ(), "EXIT_CODE="+strconv.FormatInt(tc.code, 10), "EXIT_MODE="+tc.mode)
		want := upstreamOutcome(t, dir, env, runLimit, native)
		if want.Exit != int(int32(tc.code))&255 || want.Text != fmt.Sprint("before exit ", tc.code) {
			t.Fatalf("native %d: %+v", i, want)
		}
		wants[i] = want
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			kernel := filepath.Join(out, "main.dill")
			if target == "vm" {
				run(t, out, "dart", "compile", "kernel", "--verbosity=error", "-Ddart.vm.product=false", "bin/main.dart", "-o", kernel)
			}
			for i, tc := range cases {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					env := append(os.Environ(), "EXIT_CODE="+strconv.FormatInt(tc.code, 10), "EXIT_MODE="+tc.mode)
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, dir, env, runLimit, "dart", "run", "--verbosity=error", kernel)
					} else if i == 0 {
						got = runUpstreamTargetEnv(t, target, out, "bin/main.dart", dir, []string{}, env)
					} else {
						command, args := filepath.Join(out, "tests.exe"), []string{}
						if target == "js" {
							command, args = "node", []string{filepath.Join(out, "run.js")}
						}
						if target == "wasm" {
							command, args = "node", []string{filepath.Join(out, "run.mjs")}
						}
						got = upstreamOutcome(t, dir, env, runLimit, command, args...)
					}
					if got != wants[i] {
						t.Fatalf("%s %d: Go=%+v Gart=%+v", target, i, wants[i], got)
					}
				})
			}
		})
	}
}
