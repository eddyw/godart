package driver_test

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestNodeOutput(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nodeoutput\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"os"
	"runtime"
	"time"
)

func delayedExit() { defer func() { time.Sleep(time.Millisecond); os.Exit(7) }() }
func blockedDefer() {
	defer func() { time.Sleep(time.Millisecond); println("bad defer") }()
	os.Exit(7)
}
func main() {
	mode := os.Args[1]
	defer println("main defer")
	raw := make([]byte, 1<<20)
	for i := range raw {
		raw[i] = byte(i)
	}
	print(string(raw[:256]))
	os.Stdout.Write(raw)
	os.Stderr.Write(raw)
	os.Stderr.Write(raw)
	print("en")
	println("d")
	os.Stdout.Write([]byte{0xe2})
	os.Stdout.Write([]byte{0x82, 0xac})
	os.Stderr.Write([]byte{0xf0, 0x90})
	os.Stderr.Write([]byte{0x80, 0x80})
	os.Stdout.Write([]byte("stdout tail"))
	os.Stderr.Write([]byte("stderr tail"))
	switch mode {
	case "return":
		return
	case "exit":
		os.Exit(7)
	case "exit-zero":
		os.Exit(0)
	case "blocked-defer":
		blockedDefer()
	case "from-defer":
		func() { defer os.Exit(7) }()
	case "blocked-from-defer":
		delayedExit()
	case "recover-exit":
		func() { defer func() { println("bad recover"); recover() }(); os.Exit(7) }()
	case "goroutine":
		go func() { os.Exit(7) }()
		select {}
	case "late-timer":
		time.AfterFunc(10*time.Millisecond, func() { println("bad late timer") })
		os.Exit(7)
	case "pending-return":
		started := make(chan struct{})
		go func() { close(started); time.Sleep(time.Hour); println("bad pending") }()
		<-started
		return
	case "panic":
		panic("node output panic")
	case "goexit":
		runtime.Goexit()
	default:
		panic("unknown test mode")
	}
	println("bad continuation")
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
		var stdout, stderr slowNodeOutput
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if ctx.Err() != nil || !errors.As(err, &exit) {
				t.Fatalf("execute %s: %v\n%s", name, err, stderr.buffer.String())
			}
			code = exit.ExitCode()
		}
		return result{stdout.buffer.String(), stderr.buffer.String(), code}
	}
	cases := []struct {
		name  string
		code  int
		trace bool
	}{
		{"return", 0, false}, {"exit", 7, false}, {"exit-zero", 0, false},
		{"blocked-defer", 7, false}, {"from-defer", 7, false}, {"blocked-from-defer", 7, false},
		{"recover-exit", 7, false}, {"goroutine", 7, false}, {"late-timer", 7, false},
		{"pending-return", 0, false}, {"panic", 2, true}, {"goexit", 2, true},
	}
	writeFile(t, filepath.Join(out, "runner.dart"), `import 'dart:js_interop';
import 'bin/main.dart' as app;
@JS('globalThis.gartArgs')
external JSArray<JSString> get arguments;
void main(){app.main(arguments.toDart.map((value)=>value.toDart).toList());}
`)
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var entry string
			if target == "js" {
				run(t, out, "dart", "compile", "js", "runner.dart", "-o", "main.js")
				entry = filepath.Join(out, "run.js")
				writeFile(t, entry, "globalThis.self=globalThis;globalThis.gartArgs=process.argv.slice(2);require('./main.js');\n")
			} else {
				run(t, out, "dart", "compile", "wasm", "runner.dart", "-o", "main.wasm")
				entry = filepath.Join(out, "run.mjs")
				writeFile(t, entry, `import {compile} from './main.mjs';
import {readFile} from 'node:fs/promises';
globalThis.gartArgs=process.argv.slice(2);
const app=await compile(await readFile(new URL('./main.wasm',import.meta.url)));
const instance=await app.instantiate({});
instance.invokeMain();
`)
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					want := capture(t, native, tc.name)
					if want.exit != tc.code {
						t.Fatalf("native exit=%d, want %d", want.exit, tc.code)
					}
					got := capture(t, "node", entry, tc.name)
					if tc.trace {
						want.stderr = cutTrace(want.stderr)
						got.stderr = cutTrace(got.stderr)
					}
					if got != want {
						t.Fatalf("Go: exit=%d stdout_bytes=%d stderr_bytes=%d\nGart: exit=%d stdout_bytes=%d stderr_bytes=%d", want.exit, len(want.stdout), len(want.stderr), got.exit, len(got.stdout), len(got.stderr))
					}
					t.Logf("matching exit=%d stdout_bytes=%d stderr_bytes=%d", got.exit, len(got.stdout), len(got.stderr))
				})
			}
		})
	}
}

type slowNodeOutput struct{ buffer bytes.Buffer }

func (w *slowNodeOutput) Write(p []byte) (int, error) {
	time.Sleep(time.Millisecond)
	return w.buffer.Write(p)
}
