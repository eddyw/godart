package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinBpfControls(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("BPF controls require macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/bpfcontrols\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"runtime";"syscall")
func result(name string,err error){code:=syscall.Errno(0);if err!=nil{code=err.(syscall.Errno)};fmt.Println(name,int(code))}
func empty(fd int){defer func(){v:=recover();_,typed:=v.(runtime.Error);fmt.Println("empty filter",v!=nil,typed,fmt.Sprint(v))}();syscall.SetBpf(fd,nil)}
func assess(label string,fd int){
 fmt.Println("descriptor",label)
 n,err:=syscall.BpfBuflen(fd);result("buffer",err);fmt.Println("buffer value",n)
 n,err=syscall.SetBpfBuflen(fd,(1<<40)+7);result("set buffer",err);fmt.Println("set buffer value",n)
 n,err=syscall.BpfDatalink(fd);result("datalink",err);fmt.Println("datalink value",n)
 n,err=syscall.SetBpfDatalink(fd,-1);result("set datalink",err);fmt.Println("set datalink value",n)
 result("promisc",syscall.SetBpfPromisc(fd,-1));result("flush",syscall.FlushBpf(fd))
 name,err:=syscall.BpfInterface(fd,"unchanged");result("interface",err);fmt.Println("interface value",name=="")
 result("set interface",syscall.SetBpfInterface(fd,"own-invalid-name\x00extra"))
 value,err:=syscall.BpfTimeout(fd);result("timeout",err);fmt.Println("timeout value",value==nil)
 timeout:=syscall.Timeval{Sec:-9223372036854775807,Usec:-1,Pad_cgo_0:[4]byte{1,2,3,4}};before:=timeout
 result("set timeout",syscall.SetBpfTimeout(fd,&timeout));fmt.Println("timeout unchanged",timeout==before)
 result("nil timeout",syscall.SetBpfTimeout(fd,nil))
 stats,err:=syscall.BpfStats(fd);result("stats",err);fmt.Println("stats value",stats==nil)
 result("immediate",syscall.SetBpfImmediate(fd,-1))
 instructions:=[]syscall.BpfInsn{{Code:65535,Jt:255,Jf:254,K:4294967295}};saved:=instructions[0]
 result("filter",syscall.SetBpf(fd,instructions));fmt.Println("filter unchanged",instructions[0]==saved)
 empty(fd);result("version",syscall.CheckBpfVersion(fd))
 n,err=syscall.BpfHeadercmpl(fd);result("header",err);fmt.Println("header value",n)
 result("set header",syscall.SetBpfHeadercmpl(fd,-1))
}
func main(){
 file,err:=os.CreateTemp("","godart-own-bpf-");if err!=nil{panic(err)};defer os.Remove(file.Name());defer file.Close()
 q,err:=syscall.Kqueue();if err!=nil{panic(err)};defer syscall.Close(q)
 assess("invalid",-1);assess("own file",int(file.Fd()));assess("own queue",q)
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	if want.Exit != 0 || want.Text != "descriptor invalid\nbuffer 9\nbuffer value 0\nset buffer 9\nset buffer value 0\ndatalink 9\ndatalink value 0\nset datalink 9\nset datalink value 0\npromisc 9\nflush 9\ninterface 9\ninterface value true\nset interface 9\ntimeout 9\ntimeout value true\nset timeout 9\ntimeout unchanged true\nnil timeout 14\nstats 9\nstats value true\nimmediate 9\nfilter 9\nfilter unchanged true\nempty filter true true runtime error: index out of range [0] with length 0\nversion 9\nheader 9\nheader value 0\nset header 9\ndescriptor own file\nbuffer 25\nbuffer value 0\nset buffer 25\nset buffer value 0\ndatalink 25\ndatalink value 0\nset datalink 25\nset datalink value 0\npromisc 25\nflush 25\ninterface 25\ninterface value true\nset interface 25\ntimeout 25\ntimeout value true\nset timeout 25\ntimeout unchanged true\nnil timeout 14\nstats 25\nstats value true\nimmediate 25\nfilter 25\nfilter unchanged true\nempty filter true true runtime error: index out of range [0] with length 0\nversion 25\nheader 25\nheader value 0\nset header 25\ndescriptor own queue\nbuffer 25\nbuffer value 0\nset buffer 25\nset buffer value 0\ndatalink 25\ndatalink value 0\nset datalink 25\nset datalink value 0\npromisc 25\nflush 25\ninterface 25\ninterface value true\nset interface 25\ntimeout 25\ntimeout value true\nset timeout 25\ntimeout unchanged true\nnil timeout 14\nstats 25\nstats value true\nimmediate 25\nfilter 25\nfilter unchanged true\nempty filter true true runtime error: index out of range [0] with length 0\nversion 25\nheader 25\nheader value 0\nset header 25" {
		t.Fatalf("original native BPF controls: %+v", want)
	}
	t.Log("original native BPF controls", want.Text)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("BPF controls differ: Go=%+v Gart=%+v", want, got)
			}
		})
	}
}
