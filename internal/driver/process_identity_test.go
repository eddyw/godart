package driver_test

import (
	"encoding/json"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestProcessIdentity(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("POSIX host identity")
	}
	inventory, err := parity.Discover(build.Default)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	program := `package main
import ("errors"; "encoding/json"; "os"; "slices"; "syscall")
func main() {
 group, err := syscall.Getpgid(0)
 if err != nil || group != syscall.Getpgrp() { panic("process group differs") }
 actual, err := syscall.Getpgid(syscall.Getpid())
 if err != nil || actual != group { panic("explicit process group differs") }
 queries := []func(int)(int,error){syscall.Getpgid}
 // SESSION_QUERY
 for _, query := range queries {
  for _, pid := range []int{-1, 1<<30} {
   value, err := query(pid)
   if value != -1 || !errors.Is(err,syscall.ESRCH) { panic("missing process error differs") }
  }
  value, err := query(1<<32)
  zero, zeroErr := query(0)
  if err != nil || zeroErr != nil || value != zero { panic("pid width differs") }
 }
 groups, err := os.Getgroups()
 if err != nil { panic(err) }
 rawGroups, err := syscall.Getgroups()
 if err != nil { panic(err) }
 if !slices.Equal(groups, rawGroups) || (groups == nil) != (rawGroups == nil) { panic("group identity differs") }
 if syscall.Getpid() != os.Getpid() { panic("process identity differs") }
 if syscall.Getuid() != os.Getuid() || syscall.Geteuid() != os.Geteuid() || syscall.Getgid() != os.Getgid() || syscall.Getegid() != os.Getegid() || syscall.Getppid() != os.Getppid() || syscall.Getpagesize() != os.Getpagesize() { panic("syscall identity differs") }
 json.NewEncoder(os.Stdout).Encode(append([]int{os.Getuid(), os.Geteuid(), os.Getgid(), os.Getegid(), os.Getppid(), os.Getpagesize()}, groups...))
}
`
	sessionQuery := ""
	if runtime.GOOS == "darwin" {
		sessionQuery = ` session, err := syscall.Getsid(0)
 if err != nil || session <= 0 { panic("session lookup failed") }
 actual, err = syscall.Getsid(syscall.Getpid())
 if err != nil || actual != session { panic("explicit session differs") }
 queries = append(queries, syscall.Getsid)
`
	}
	writeFile(t, filepath.Join(dir, "main.go"), strings.Replace(program, " // SESSION_QUERY\n", sessionQuery, 1))

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processidentity\n\ngo 1.27\n\nrequire example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/exec_unix_test.go"), []string{"command", "command.Info", "command.Start", "command.Stop", "create", "parent", "TestZeroSysProcAttr"})
	source = strings.Replace(source, "package syscall_test", "package main", 1)
	source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	if err := os.Mkdir(filepath.Join(dir, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	helper := selectedUpstreamSource(t, "../overlay/std/internal/testenv/testenv.go", []string{"MustHaveExec"})
	writeFile(t, filepath.Join(dir, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "testenv/testenv.go"), strings.TrimPrefix(helper, "//go:build dart\n"))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "testenv/LICENSE"), string(license))

	native := filepath.Join(dir, "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := runProgram(t, dir, native)
	var observed []int
	if err := json.Unmarshal([]byte(want), &observed); err != nil {
		t.Fatal(err)
	}
	if len(observed) < 6 || observed[4] != os.Getpid() {
		t.Fatalf("native parent contract: %v", observed)
	}
	t.Run("node", func(t *testing.T) { testNodeIdentity(t, observed[:4]) })
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	t.Run("vm", func(t *testing.T) {
		if got := runProgram(t, out, "dart", "bin/main.dart"); got != want {
			t.Errorf("identity mismatch: Go %s Dart %s", want, got)
		}
	})
	t.Run("aot", func(t *testing.T) {
		exe := filepath.Join(out, "identity")
		run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
		if got := runProgram(t, out, exe); got != want {
			t.Errorf("identity mismatch: Go %s Dart %s", want, got)
		}
	})
	testOut := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: testOut, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, testOut)
	for _, target := range []string{"vm", "aot"} {
		t.Run("upstream/"+target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=^TestZeroSysProcAttr$")
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(testOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, testOut, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "syscall-process-group", "syscall", "^TestZeroSysProcAttr$", target, dir, result.Text, result.Exit, *inventory, nil, nil, nil, false)
		})
	}
}

func testNodeIdentity(t *testing.T, native []int) {
	out := t.TempDir()
	writeFile(t, filepath.Join(out, "pubspec.yaml"), fmt.Sprintf("name: node_identity\nenvironment:\n  sdk: ^3.13.0\ndependencies:\n  goruntime:\n    path: %q\n", runtimeDir(t)))
	writeFile(t, filepath.Join(out, "main.dart"), `import 'dart:convert';
import 'package:goruntime/src/identity_web.dart';
void main() => print(jsonEncode([$getuid(), $geteuid(), $getgid(), $getegid()]));
`)
	run(t, out, "dart", "pub", "get")
	cases := []struct {
		name, setup string
		want        []int
	}{
		{"native", "", native},
		{"absent", "globalThis.process=undefined;", []int{-1, -1, -1, -1}},
		{"missing", "globalThis.process={};", []int{-1, -1, -1, -1}},
	}
	methods := []string{"getuid", "geteuid", "getgid", "getegid"}
	for missing := -1; missing < len(methods); missing++ {
		name := "distinct"
		if missing >= 0 {
			name = "missing-" + methods[missing]
		}
		fields := []string{"marker:true"}
		want := []int{101, 202, 303, 404}
		for i, method := range methods {
			if i == missing {
				want[i] = -1
				continue
			}
			fields = append(fields, fmt.Sprintf("%s(){if(this.marker!==true)throw Error('receiver');return %d;}", method, want[i]))
		}
		cases = append(cases, struct {
			name, setup string
			want        []int
		}{name, "globalThis.process={" + strings.Join(fields, ",") + "};", want})
	}
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			run(t, out, "dart", "compile", target, "main.dart", "-o", "main."+target)
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					entry := "run.js"
					source := "globalThis.self=globalThis;\n" + tc.setup + "\nrequire('./main.js');\n"
					if target == "wasm" {
						entry = "run.mjs"
						source = `import {compile} from './main.mjs';
import {readFile} from 'node:fs/promises';
const app=await compile(await readFile(new URL('./main.wasm',import.meta.url)));
const instance=await app.instantiate({});
` + tc.setup + "\ninstance.invokeMain();\n"
					}
					writeFile(t, filepath.Join(out, entry), source)
					want, err := json.Marshal(tc.want)
					if err != nil {
						t.Fatal(err)
					}
					got := programOutcome(t, out, "node", entry)
					if got.Exit != 0 || got.Text != string(want) {
						t.Fatalf("identity contract differs: %+v", got)
					}
				})
			}
		})
	}
}
