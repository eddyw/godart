package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinRouteRIB(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/routerib\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"bytes"
 "errors"
 "os"
	"fmt"
	"syscall"
)

func messages(data []byte) bool {
	for len(data) >= 4 {
		n := int(data[0]) | int(data[1])<<8
		if n < 4 || n > len(data) || data[2] != syscall.RTM_VERSION {
			return false
		}
		data = data[n:]
	}
	return len(data) == 0
}
func main() {
 if len(os.Args)>1 {
  for i,values:=range [][2]int{{1,2},{2,0},{-1,0},{999,0},{1+(1<<32),2+(1<<32)}} {
   data,err:=syscall.RouteRIB(values[0],values[1]);code,typed:=err.(syscall.Errno)
   fmt.Println("unavailable",i,data==nil,len(data),int(code),typed,errors.Is(err,errors.ErrUnsupported))
  }
  return
 }

	for _, values := range [][2]int{{syscall.NET_RT_DUMP, syscall.AF_INET}, {syscall.NET_RT_DUMP, syscall.AF_INET6}, {syscall.NET_RT_IFLIST, 0}, {syscall.NET_RT_FLAGS, 0}, {syscall.NET_RT_FLAGS, 1 << 30}, {-1, 0}, {999, 0}, {syscall.NET_RT_DUMP, -1}, {syscall.NET_RT_DUMP, 999}, {syscall.NET_RT_DUMP + (1 << 32), syscall.AF_INET + (1 << 32)}} {
		data, err := syscall.RouteRIB(values[0], values[1])
		code, typed := err.(syscall.Errno)
		fmt.Println("query", values[0], values[1], int(code), typed, data == nil, len(data) > 0, messages(data))
		if err != nil || len(data) == 0 {
			continue
		}
		other, err := syscall.RouteRIB(values[0], values[1])
		if err != nil {
			panic(err)
		}
		copy := append([]byte(nil), other...)
		alias := data
		old := data[0]
		alias[0] ^= 255
		fmt.Println("ownership", len(other) > 0, &data[0] != &other[0], bytes.Equal(other, copy), data[0] == old^255, messages(other))
	}
}
`)
	var want programResult
	if runtime.GOOS == "darwin" {
		native := filepath.Join(t.TempDir(), "native")
		run(t, dir, "go", "build", "-o", native, ".")
		want = upstreamOutcome(t, dir, os.Environ(), runLimit, native)
		if want.Exit != 0 || want.Text != "query 1 2 0 false false true true\nownership true true true true true\nquery 1 30 0 false false true true\nownership true true true true true\nquery 3 0 0 false false true true\nownership true true true true true\nquery 2 0 0 false true false true\nquery 2 1073741824 0 false false true true\nownership true true true true true\nquery -1 0 22 true true false true\nquery 999 0 22 true true false true\nquery 1 -1 0 false false true true\nownership true true true true true\nquery 1 999 0 false false true true\nownership true true true true true\nquery 4294967297 4294967298 0 false false true true\nownership true true true true true" {
			t.Fatalf("native kernel query: %+v", want)
		}
		t.Log("original SDK RouteRIB", want.Text)
	}
	t.Setenv("GOOS", "darwin")
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			t.Setenv("GOARCH", arch)
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			if runtime.GOOS == "darwin" {
				for _, target := range []string{"vm", "aot"} {
					t.Run(target, func(t *testing.T) {
						var got programResult
						if target == "vm" {
							got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", filepath.Join(out, "bin/main.dart"))
						} else {
							got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
						}
						if got != want {
							t.Fatalf("kernel query differs: Go=%+v Gart=%+v", want, got)
						}
					})
				}
			}
			for _, target := range []string{"js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unavailable"})
					if got.Exit != 0 || got.Text != "unavailable 0 true 0 78 true true\nunavailable 1 true 0 78 true true\nunavailable 2 true 0 78 true true\nunavailable 3 true 0 78 true true\nunavailable 4 true 0 78 true true" {
						t.Fatalf("web query differs: %+v", got)
					}
				})
			}
		})
	}
}
