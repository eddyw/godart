package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxSendfile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Sendfile requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"syscall")
func must(err error){if err!=nil{panic(err)}}
func position(fd int)int64{n,err:=syscall.Seek(fd,0,1);must(err);return n}
func main(){
 dir,err:=os.MkdirTemp("","sendfile");must(err);defer os.RemoveAll(dir)
 path:=dir+"/source";payload:=[]byte("ab\x00cd\xffefghij");must(os.WriteFile(path,payload,0600))
 input,err:=syscall.Open(path,syscall.O_RDONLY,0);must(err);defer syscall.Close(input)
 output,err:=syscall.Open(dir+"/output",syscall.O_RDWR|syscall.O_CREAT|syscall.O_TRUNC,0600);must(err);defer syscall.Close(output)
 n,err:=syscall.Sendfile(output,input,nil,3);must(err);fmt.Println("nil",n,position(input),position(output))
 offset:=int64(5);n,err=syscall.Sendfile(output,input,&offset,4);must(err);fmt.Println("explicit",n,offset,position(input),position(output))
 duplicate,err:=syscall.Dup(input);must(err);defer syscall.Close(duplicate)
 n,err=syscall.Sendfile(output,duplicate,nil,2);must(err);fmt.Println("duplicate",n,position(input),position(duplicate))
 offset=int64(len(payload)-2);n,err=syscall.Sendfile(output,input,&offset,9);must(err);fmt.Println("partial EOF",n,offset,position(input))
 n,err=syscall.Sendfile(output,input,&offset,9);must(err);fmt.Println("EOF",n,offset)
 n,err=syscall.Sendfile(output,input,&offset,0);must(err);fmt.Println("zero",n,offset)
 data,err:=os.ReadFile(dir+"/output");must(err);fmt.Printf("bytes %x\n",data)
 pair,err:=syscall.Socketpair(syscall.AF_UNIX,syscall.SOCK_STREAM,0);must(err);defer syscall.Close(pair[0]);defer syscall.Close(pair[1])
 offset=1;n,err=syscall.Sendfile(pair[0],input,&offset,7);must(err)
 buffer:=make([]byte,7);read,err:=syscall.Read(pair[1],buffer);must(err);fmt.Printf("socket %d %d %d %x\n",n,read,offset,buffer[:read])
 offset=-1;n,err=syscall.Sendfile(output,input,&offset,1);fmt.Println("negative offset",n,err==syscall.EINVAL,offset)
 offset=2;n,err=syscall.Sendfile(-1,input,&offset,1);fmt.Println("bad output",n,err==syscall.EBADF,offset)
 n,err=syscall.Sendfile(output,-1,&offset,1);fmt.Println("bad input",n,err==syscall.EBADF,offset)
 n,err=syscall.Sendfile(output,input,&offset,-1);fmt.Println("negative count",n,err==syscall.EINVAL,offset)
 sparse,err:=syscall.Open(dir+"/sparse",syscall.O_RDWR|syscall.O_CREAT,0600);must(err);defer syscall.Close(sparse)
 high:=int64(1<<33|19);w,err:=syscall.Pwrite(sparse,[]byte("wide"),high);must(err);if w!=4{panic("sparse write")}
 offset=high;n,err=syscall.Sendfile(pair[0],sparse,&offset,4);must(err);read,err=syscall.Read(pair[1],buffer[:4]);must(err)
 fmt.Println("wide",n,offset==high+4,position(sparse),string(buffer[:read]))
 must(syscall.Close(input));n,err=syscall.Sendfile(pair[0],duplicate,nil,1);must(err);read,err=syscall.Read(pair[1],buffer[:1]);must(err)
 fmt.Println("owner",n,read,string(buffer[:read]),position(duplicate))
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
