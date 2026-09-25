package driver_test

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestNodeOutput(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nodeoutput\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "os"
func main(){
 raw:=make([]byte,256)
 for i:=range raw {raw[i]=byte(i)}
 os.Stdout.Write(raw)
 os.Stderr.Write(raw)
 print(string(raw));println("end")
 os.Stdout.Write([]byte{0xe2});os.Stdout.Write([]byte{0x82,0xac})
 os.Stderr.Write([]byte{0xf0,0x90});os.Stderr.Write([]byte{0x80,0x80})
 os.Stdout.Write([]byte("stdout tail"))
 os.Stderr.Write([]byte("stderr tail"))
 if len(os.Args)>1 {os.Exit(7)}
}
`)
	native := filepath.Join(t.TempDir(), "program")
	run(t, dir, "go", "build", "-o", native, ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	type result struct {
		stdout, stderr string
		exit           int
	}
	capture := func(t *testing.T, name string, args ...string) result {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if ctx.Err() != nil || !errors.As(err, &exit) {
				t.Fatalf("execute %s: %v\n%s", name, err, stderr.String())
			}
			code = exit.ExitCode()
		}
		return result{stdout.String(), stderr.String(), code}
	}
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			for _, code := range []int{0, 7} {
				t.Run(strconv.Itoa(code), func(t *testing.T) {
					args := []string{}
					dartArgs := "<String>[]"
					if code != 0 {
						args = []string{"exit"}
						dartArgs = "<String>['exit']"
					}
					want := capture(t, native, args...)
					if want.exit != code {
						t.Fatalf("native exit=%d, want %d", want.exit, code)
					}
					writeFile(t, filepath.Join(out, "runner.dart"), "import 'bin/main.dart' as app;\nvoid main(){app.main("+dartArgs+");}\n")
					var entry string
					if target == "js" {
						run(t, out, "dart", "compile", "js", "runner.dart", "-o", "main.js")
						entry = filepath.Join(out, "run.js")
						writeFile(t, entry, "globalThis.self=globalThis;\nrequire('./main.js');\n")
					} else {
						run(t, out, "dart", "compile", "wasm", "runner.dart", "-o", "main.wasm")
						entry = filepath.Join(out, "run.mjs")
						writeFile(t, entry, `import {compile} from './main.mjs';
import {readFile} from 'node:fs/promises';
const app=await compile(await readFile(new URL('./main.wasm',import.meta.url)));
const instance=await app.instantiate({});
instance.invokeMain();
`)
					}
					got := capture(t, "node", entry)
					if got != want {
						t.Fatalf("Go: exit=%d stdout=%x stderr=%x\nGart: exit=%d stdout=%x stderr=%x", want.exit, want.stdout, want.stderr, got.exit, got.stdout, got.stderr)
					}
					t.Logf("matching exit=%d stdout_bytes=%d stderr_bytes=%d", got.exit, len(got.stdout), len(got.stderr))
				})
			}
		})
	}
}
