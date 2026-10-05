package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLinuxPtraceValues(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("event value reference requires a selected 64-bit host")
	}
	const source = `package main
import("fmt";"runtime";"syscall")
func nilValue(name string, call func()) {
 defer func(){value:=recover();_,typed:=value.(runtime.Error);fmt.Println("nil",name,value!=nil,typed,fmt.Sprint(value))}()
 call()
}
func main(){
 original := __REG_INIT__
 for i,pc:=range []uint64{0,1,4294967295,4294967296,9223372036854775807,9223372036854775808,18446744073709551614,18446744073709551615}{
  value:=original
  alias:=&value
  value.SetPC(pc)
  fmt.Println("pc",i,value.PC(),alias.PC(),value.__PC_FIELD__,value.PC()==pc)
  value.__PC_FIELD__=original.__PC_FIELD__
  fmt.Println("retained",i,value==original,alias==&value,original.PC())
 }
 nilValue("PC",func(){var value *syscall.PtraceRegs;value.PC()})
 nilValue("SetPC",func(){var value *syscall.PtraceRegs;value.SetPC(1)})
 fmt.Println("original",original.PC())
}
`
	initial := map[string]string{"amd64": "syscall.PtraceRegs{R15:1,R14:2,R13:3,R12:4,Rbp:5,Rbx:6,R11:7,R10:8,R9:9,R8:10,Rax:11,Rcx:12,Rdx:13,Rsi:14,Rdi:15,Orig_rax:16,Rip:17,Cs:18,Eflags:19,Rsp:20,Ss:21,Fs_base:22,Gs_base:23,Ds:24,Es:25,Fs:26,Gs:27}", "arm64": "syscall.PtraceRegs{Regs:[31]uint64{1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31},Sp:41,Pc:17,Pstate:43}"}
	field := map[string]string{"amd64": "Rip", "arm64": "Pc"}
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			fixture := strings.NewReplacer("__REG_INIT__", initial[arch], "__PC_FIELD__", field[arch]).Replace(source)
			reference, dir, out := t.TempDir(), t.TempDir(), t.TempDir()
			if err := os.Mkdir(filepath.Join(reference, "reference"), 0755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(reference, "go.mod"), "module example.com/eventreference\n\ngo 1.27\n")
			writeFile(t, filepath.Join(reference, "reference", "types.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/ztypes_linux_"+arch+".go"), []string{"PtraceRegs"}))
			writeFile(t, filepath.Join(reference, "reference", "setter.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_linux_"+arch+".go"), []string{"PtraceRegs.PC", "PtraceRegs.SetPC"}))
			writeFile(t, filepath.Join(reference, "main.go"), strings.Replace(fixture, "\"syscall\"", "syscall \"example.com/eventreference/reference\"", 1))
			native := filepath.Join(t.TempDir(), "native")
			run(t, reference, "go", "build", "-o", native, ".")
			want := upstreamOutcome(t, reference, os.Environ(), runLimit, native)
			if want.Exit != 0 || want.Text != "pc 0 0 0 0 true\nretained 0 true true 17\npc 1 1 1 1 true\nretained 1 true true 17\npc 2 4294967295 4294967295 4294967295 true\nretained 2 true true 17\npc 3 4294967296 4294967296 4294967296 true\nretained 3 true true 17\npc 4 9223372036854775807 9223372036854775807 9223372036854775807 true\nretained 4 true true 17\npc 5 9223372036854775808 9223372036854775808 9223372036854775808 true\nretained 5 true true 17\npc 6 18446744073709551614 18446744073709551614 18446744073709551614 true\nretained 6 true true 17\npc 7 18446744073709551615 18446744073709551615 18446744073709551615 true\nretained 7 true true 17\nnil PC true true runtime error: invalid memory address or nil pointer dereference\nnil SetPC true true runtime error: invalid memory address or nil pointer dereference\noriginal 17" {
				t.Fatalf("original SDK ptrace values: %+v", want)
			}
			t.Log("original SDK ptrace values", arch, want.Text)
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/eventvalues\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "main.go"), fixture)
			t.Setenv("GOOS", "linux")
			t.Setenv("GOARCH", arch)
			t.Setenv("CGO_ENABLED", "0")
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			for _, target := range []string{"vm", "aot", "js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", filepath.Join(out, "bin/main.dart"))
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
					}
					if got != want {
						t.Fatalf("SDK ptrace values differ: Go=%+v Gart=%+v", want, got)
					}
				})
			}
		})
	}
}
