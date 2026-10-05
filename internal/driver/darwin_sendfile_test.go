package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinSendfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/darwinsendfile\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"os";"syscall")
func must(err error){if err!=nil{panic(err)}}
func pos(fd int)int64{n,err:=syscall.Seek(fd,0,1);must(err);return n}
func nilOffset(out,in int){defer func(){fmt.Println("nil panic",recover()!=nil)}();syscall.Sendfile(out,in,nil,1)}
func main(){
 if len(os.Args)>1 {
  for i,v:=range [][2]int64{{1,7},{-1,-1},{1<<33|19,1<<33|7}}{
   offset:=v[0];n,err:=syscall.Sendfile(-1,-1,&offset,int(v[1]));code,typed:=err.(syscall.Errno)
   fmt.Println("unavailable",i,n,int(code),typed,errors.Is(err,errors.ErrUnsupported),offset==v[0])
  }
  nilOffset(-1,-1)
  return
 }

 dir,err:=os.MkdirTemp("","darwin-sendfile");must(err);defer os.RemoveAll(dir)
 payload:=[]byte("ab\x00cd\xffefghij");must(os.WriteFile(dir+"/input",payload,0600))
 input,err:=syscall.Open(dir+"/input",syscall.O_RDONLY,0);must(err);defer syscall.Close(input)
 pair,err:=syscall.Socketpair(syscall.AF_UNIX,syscall.SOCK_STREAM,0);must(err);defer syscall.Close(pair[0]);defer syscall.Close(pair[1])
 must(syscall.SetNonblock(pair[0],true));must(syscall.SetNonblock(pair[1],true))
 buffer:=make([]byte,128)
 for _,v:=range [][2]int64{{1,7},{10,9},{12,9},{2,0},{-1,1},{2,-1}}{
  offset:=v[0];n,err:=syscall.Sendfile(pair[0],input,&offset,int(v[1]));code,typed:=err.(syscall.Errno)
  fmt.Println("call",v,n,int(code),typed,offset==v[0],pos(input))
  if n>0{read,err:=syscall.Read(pair[1],buffer[:n]);must(err);fmt.Printf("bytes %d %x\n",read,buffer[:read])}
 }
 for _,v:=range [][2]int{{-1,input},{pair[0],-1},{input,input}}{offset:=int64(2);n,err:=syscall.Sendfile(v[0],v[1],&offset,3);code,typed:=err.(syscall.Errno);fmt.Println("error",n,int(code),typed,offset)}
 high:=int64(1<<33|19);sparse,err:=syscall.Open(dir+"/sparse",syscall.O_RDWR|syscall.O_CREAT,0600);must(err);defer syscall.Close(sparse)
 n,err:=syscall.Pwrite(sparse,[]byte("wide"),high);must(err);if n!=4{panic("write")};offset:=high;n,err=syscall.Sendfile(pair[0],sparse,&offset,4);must(err);read,err:=syscall.Read(pair[1],buffer[:n]);must(err);fmt.Println("wide",n,offset==high,pos(sparse),string(buffer[:read]))
 duplicate,err:=syscall.Dup(input);must(err);defer syscall.Close(duplicate);must(syscall.Close(input));offset=3;n,err=syscall.Sendfile(pair[0],duplicate,&offset,2);must(err);read,err=syscall.Read(pair[1],buffer[:n]);must(err);fmt.Printf("owner %d %d %x %d\n",n,offset,buffer[:read],pos(duplicate))

 offset=1;n,err=syscall.Sendfile(pair[0]+(1<<32),duplicate+(1<<32),&offset,3);must(err);read,err=syscall.Read(pair[1],buffer[:n]);must(err);fmt.Printf("fd widths %d %d %x\n",n,offset,buffer[:read])
 large:=make([]byte,1<<20);for i:=range large{large[i]=byte(i*17+3)};must(os.WriteFile(dir+"/large",large,0600));lfd,err:=syscall.Open(dir+"/large",syscall.O_RDONLY,0);must(err);defer syscall.Close(lfd)
 must(syscall.SetsockoptInt(pair[0],syscall.SOL_SOCKET,syscall.SO_SNDBUF,1024));offset=0;n,err=syscall.Sendfile(pair[0],lfd,&offset,len(large));fmt.Println("partial",n>0,n<len(large),err==syscall.EAGAIN,offset==0,pos(lfd)==0)
 received:=make([]byte,n);total:=0;for total<n{r,err:=syscall.Read(pair[1],received[total:]);must(err);if r==0{panic("short socket")};total+=r};fmt.Println("partial bytes",bytes.Equal(received,large[:n]))

 _,err=syscall.Seek(duplicate,5,0);must(err);offset=1;n,err=syscall.Sendfile(pair[0],duplicate,&offset,2);must(err);read,err=syscall.Read(pair[1],buffer[:n]);must(err);fmt.Printf("position %d %d %d %x\n",n,offset,pos(duplicate),buffer[:read])
 offset=1;n,err=syscall.Sendfile(pair[0],duplicate,&offset,(1<<32)+2);must(err);read,err=syscall.Read(pair[1],buffer[:n]);must(err);fmt.Printf("count width %d %d %d %x\n",n,offset,pos(duplicate),buffer[:read])
 for _,count:=range []int{-1,1<<33|7}{offset=2;n,err=syscall.Sendfile(-1,duplicate,&offset,count);code,typed:=err.(syscall.Errno);fmt.Println("bad count",count,n,int(code),typed,offset)}
 nilOffset(pair[0],duplicate)
}
`)
	var want programResult
	if runtime.GOOS == "darwin" {
		native := filepath.Join(t.TempDir(), "native")
		run(t, dir, "go", "build", "-o", native, ".")
		want = upstreamOutcome(t, dir, os.Environ(), runLimit, native)
		if want.Exit != 0 || want.Text != "call [1 7] 7 0 false true 0\nbytes 7 62006364ff6566\ncall [10 9] 2 0 false true 0\nbytes 2 696a\ncall [12 9] 0 0 false true 0\ncall [2 0] 10 0 false true 0\nbytes 10 006364ff65666768696a\ncall [-1 1] 0 22 true true 0\ncall [2 -1] 0 0 false true 0\nerror 0 9 true 2\nerror 0 9 true 2\nerror 0 38 true 2\nwide 4 true 0 wide\nowner 2 3 6364 0\nfd widths 3 1 620063\npartial true true true true true\npartial bytes true\nposition 2 1 5 6200\ncount width 11 1 5 62006364ff65666768696a\nbad count -1 0 9 true 2\nbad count 8589934599 0 9 true 2\nnil panic true" {
			t.Fatalf("native transfer: %+v", want)
		}
		t.Log("original SDK Darwin Sendfile", want.Text)
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
							t.Fatalf("native transfer differs: Go=%+v Gart=%+v", want, got)
						}
					})
				}
			}
			for _, target := range []string{"js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unavailable"})
					if got.Exit != 0 || got.Text != "unavailable 0 0 78 true true true\nunavailable 1 0 78 true true true\nunavailable 2 0 78 true true true\nnil panic true" {
						t.Fatalf("web transfer differs: %+v", got)
					}
				})
			}
		})
	}
}
