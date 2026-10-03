package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeUnixSendtoOwnership(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Raw Unix datagrams require native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unixsendto\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"net";"os";"path/filepath";"strings";"syscall";"time")
func main(){
 dir,err:=os.MkdirTemp("","us");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 name:=filepath.Join(dir,"s");server,err:=net.ListenUnixgram("unixgram",&net.UnixAddr{Name:name,Net:"unixgram"});if err!=nil{panic(err)};defer server.Close()
 fd,err:=syscall.Socket(syscall.AF_UNIX,syscall.SOCK_DGRAM,0);if err!=nil{panic(err)}
 var destination syscall.Sockaddr=&syscall.SockaddrUnix{Name:name}
 payload:=[]byte{0,255,128,1,2}
 for _,data:=range [][]byte{payload,nil,payload}{
  if err:=syscall.Sendto(fd,data,syscall.MSG_DONTWAIT,destination);err!=nil{panic(err)}
  if err:=server.SetReadDeadline(time.Now().Add(time.Second));err!=nil{panic(err)}
  b:=make([]byte,16);n,from,err:=server.ReadFromUnix(b)
  if err!=nil||from!=nil||!bytes.Equal(data,b[:n]){panic("raw bytes or sender differ")}
 }
 fmt.Println("packets and sender",true)
 fmt.Println("no destination",syscall.Sendto(fd,nil,syscall.MSG_DONTWAIT,nil)!=nil)
 fmt.Println("long address",errors.Is(syscall.Sendto(fd,nil,0,&syscall.SockaddrUnix{Name:strings.Repeat("x",109)}),syscall.EINVAL))
 fmt.Println("ownership",syscall.Close(fd)==nil)
 fmt.Println("closed descriptor",errors.Is(syscall.Sendto(fd,payload,0,destination),syscall.EBADF))
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
