package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSyscallByteIO(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Raw byte I/O requires native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rawbyteio\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"os";"syscall")
func result(name string,n int,err error){fmt.Println(name,n,err==nil,errors.Is(err,syscall.EBADF),errors.Is(err,syscall.EAGAIN))}
func main(){
 f,err:=os.CreateTemp("","rawbyte");if err!=nil{panic(err)};name:=f.Name();if err=f.Close();err!=nil{panic(err)};defer os.Remove(name)
 fd,err:=syscall.Open(name,syscall.O_RDWR,0600);if err!=nil{panic(err)}
 data:=[]byte{99,0,255,128,1,99};n,err:=syscall.Write(fd,data[1:5]);result("file write",n,err)
 _,err=syscall.Seek(fd,0,0);if err!=nil{panic(err)}
 b:=bytes.Repeat([]byte{99},8);n,err=syscall.Read(fd,b[2:6]);result("file read",n,err);fmt.Println("slice bytes",bytes.Equal(b,[]byte{99,99,0,255,128,1,99,99}))
 n,err=syscall.Read(fd,b);result("file eof",n,err)
 n,err=syscall.Read(fd,nil);result("file empty read",n,err)
 n,err=syscall.Write(fd,nil);result("file empty write",n,err)
 if err=syscall.Close(fd);err!=nil{panic(err)}
 n,err=syscall.Read(fd,b);result("file closed read",n,err)
 n,err=syscall.Write(fd,data);result("file closed write",n,err)
 r,w,err:=os.Pipe();if err!=nil{panic(err)};defer r.Close();defer w.Close()
 rfd,wfd:=int(r.Fd()),int(w.Fd());if err=syscall.SetNonblock(rfd,true);err!=nil{panic(err)};if err=syscall.SetNonblock(wfd,true);err!=nil{panic(err)}
 n,err=syscall.Read(rfd,b);result("pipe empty",n,err)
 n,err=syscall.Read(wfd,b);result("pipe wrong read",n,err)
 n,err=syscall.Write(rfd,data);result("pipe wrong write",n,err)
 n,err=syscall.Write(wfd,data[1:5]);result("pipe write",n,err)
 b=make([]byte,8);n,err=syscall.Read(rfd,b[2:6]);result("pipe read",n,err);fmt.Println("pipe bytes",bytes.Equal(b,[]byte{0,0,0,255,128,1,0,0}))
 chunk:=bytes.Repeat([]byte{7},4096);total:=0
 for {n,err=syscall.Write(wfd,chunk);if errors.Is(err,syscall.EAGAIN){break};if err!=nil||n<=0{panic("pipe fill failed")};total+=n}
 n,err=syscall.Write(wfd,chunk);result("pipe full",n,err)
 n,err=syscall.Write(wfd,nil);result("pipe full empty",n,err)
 drained:=0
 for drained<total {n,err=syscall.Read(rfd,chunk);if err!=nil||n<=0{panic("pipe drain failed")};for _,v:=range chunk[:n]{if v!=7{panic("pipe contents differ")}};drained+=n}
 fmt.Println("pipe fill and drain",total>0&&drained==total)
 if err=w.Close();err!=nil{panic(err)};n,err=syscall.Read(rfd,b);result("pipe eof",n,err)
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
