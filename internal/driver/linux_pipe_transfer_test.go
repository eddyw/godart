package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxPipeTransfer(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Splice and Tee require Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"syscall";"strconv";"strings")
func must(err error){if err!=nil{panic(err)}}
func pipe()[]int{p:=make([]int,2);must(syscall.Pipe2(p,syscall.O_NONBLOCK));return p}
func read(fd,n int)[]byte{b:=make([]byte,n);count,err:=syscall.Read(fd,b);must(err);return b[:count]}
func position(fd int)int64{n,err:=syscall.Seek(fd,0,1);must(err);return n}
func main(){
 a,b:=pipe(),pipe();defer syscall.Close(a[0]);defer syscall.Close(a[1]);defer syscall.Close(b[0]);defer syscall.Close(b[1])
 for _,flags:=range []int{0,syscall.O_NONBLOCK,syscall.O_CLOEXEC,syscall.O_NONBLOCK|syscall.O_CLOEXEC} {
  pair:=make([]int,2);must(syscall.Pipe2(pair,flags))
  for _,fd:=range pair {
   data,err:=os.ReadFile(fmt.Sprintf("/proc/self/fdinfo/%d",fd));must(err)
   found:=false
   for _,line:=range strings.Split(string(data),"\n") {if strings.HasPrefix(line,"flags:") {bits,err:=strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line,"flags:")),8,64);must(err);fmt.Println("pipe flags",bits&syscall.O_NONBLOCK!=0,bits&syscall.O_CLOEXEC!=0);found=true}}
   if !found {panic("missing fdinfo flags")}
  }
  must(syscall.Close(pair[0]));must(syscall.Close(pair[1]))
 }
 for _,size:=range []int{0,1,3} {p:=[]int{a[0],a[1],b[0]}[:size];before:=fmt.Sprint(p);err:=syscall.Pipe2(p,0);fmt.Println("pipe length",size,err==syscall.EINVAL,fmt.Sprint(p)==before)}
 unchanged:=[]int{a[0],a[1]};err:=syscall.Pipe2(unchanged,1<<30);fmt.Println("pipe invalid",err==syscall.EINVAL,unchanged[0]==a[0],unchanged[1]==a[1])
 payload:=[]byte("ab\x00cd\xffefghij");w,err:=syscall.Write(a[1],payload);must(err);if w!=len(payload){panic("write")}
 n,err:=syscall.Tee(a[0],b[1],4,2);must(err);fmt.Printf("tee %d %x\n",n,read(b[0],4))
 n,err=syscall.Splice(a[0],nil,b[1],nil,3,2);must(err);fmt.Printf("splice %d %x %x\n",n,read(b[0],3),read(a[0],len(payload)-3))
 n,err=syscall.Tee(a[0],b[1],1,2);fmt.Println("tee empty",n,err==syscall.EAGAIN)
 n,err=syscall.Splice(a[0],nil,b[1],nil,1,2);fmt.Println("splice empty",n,err==syscall.EAGAIN)
 dir,err:=os.MkdirTemp("","splice");must(err);defer os.RemoveAll(dir);must(os.WriteFile(dir+"/source",payload,0600))
 input,err:=syscall.Open(dir+"/source",syscall.O_RDONLY,0);must(err);defer syscall.Close(input)
 output,err:=syscall.Open(dir+"/output",syscall.O_RDWR|syscall.O_CREAT|syscall.O_TRUNC,0600);must(err);defer syscall.Close(output)
 roff,woff:=int64(1),int64(4)
 n,err=syscall.Splice(input,&roff,b[1],nil,5,2);must(err);fmt.Println("file input",n,roff,position(input))
 n,err=syscall.Splice(b[0],nil,output,&woff,5,2);must(err);fmt.Println("file output",n,woff,position(output));data,err:=os.ReadFile(dir+"/output");must(err);fmt.Printf("file bytes %x\n",data)
 n,err=syscall.Splice(input,nil,b[1],nil,2,0);must(err);fmt.Printf("nil input %d %d %x\n",n,position(input),read(b[0],2))
 w,err=syscall.Write(a[1],[]byte("XY"));must(err);if w!=2{panic("write nil")};n,err=syscall.Splice(a[0],nil,output,nil,2,0);must(err);fmt.Println("nil output",n,position(output))
 shared:=int64(3);n,err=syscall.Splice(a[0],&shared,b[1],&shared,1,2);fmt.Println("shared pipe offset",n,err==syscall.ESPIPE,shared)
 roff=-1;n,err=syscall.Splice(input,&roff,b[1],nil,1,2);fmt.Println("negative offset",n,err==syscall.EINVAL,roff)
 n,err=syscall.Splice(-1,nil,b[1],nil,1,2);fmt.Println("splice bad",n,err==syscall.EBADF)
 n,err=syscall.Tee(-1,b[1],1,2);fmt.Println("tee bad",n,err==syscall.EBADF)
 n,err=syscall.Tee(a[0],b[1],1,1<<30);fmt.Println("tee flags",n,err==syscall.EINVAL)
 n,err=syscall.Splice(input,nil,b[1],nil,1,1<<30);fmt.Println("splice flags",n,err==syscall.EINVAL)
 n,err=syscall.Splice(input,nil,b[1],nil,-1,2);fmt.Println("splice count",n,err==syscall.EINVAL)
 n,err=syscall.Tee(a[0],b[1],-1,2);fmt.Println("tee count",n,err==syscall.EINVAL)
 n,err=syscall.Tee(a[0],b[1],0,2);must(err);fmt.Println("tee zero",n)
 n,err=syscall.Splice(a[0],nil,b[1],nil,0,2);must(err);fmt.Println("splice zero",n)
 sparse,err:=syscall.Open(dir+"/sparse",syscall.O_RDWR|syscall.O_CREAT,0600);must(err);defer syscall.Close(sparse)
 high:=int64(1<<33|19);w,err=syscall.Pwrite(sparse,[]byte("wide"),high);must(err);if w!=4{panic("wide write")}
 roff=high;n,err=syscall.Splice(sparse,&roff,b[1],nil,4,2);must(err);fmt.Println("wide",n,roff==high+4,position(sparse),string(read(b[0],4)))
 duplicate,err:=syscall.Dup(a[1]);must(err);must(syscall.Close(a[1]));w,err=syscall.Write(duplicate,[]byte("tail"));must(err);if w!=4{panic("tail write")};must(syscall.Close(duplicate))
 n,err=syscall.Tee(a[0],b[1],4,2);must(err);fmt.Println("owner tee",n,string(read(b[0],4)),string(read(a[0],4)))
 n,err=syscall.Tee(a[0],b[1],1,2);must(err);fmt.Println("tee EOF",n)
 n,err=syscall.Splice(a[0],nil,b[1],nil,1,2);must(err);fmt.Println("splice EOF",n)
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
