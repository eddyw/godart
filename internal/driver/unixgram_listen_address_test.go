package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnixgramListenAddressContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix listener addresses require a native platform")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unixgramaddress\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"net";"runtime";"syscall";"time")
func main(){
 _,err:=net.ListenUnixgram("unixgram",nil);fmt.Println("nil address",err!=nil)
 server,err:=net.ListenUnixgram("unixgram",&net.UnixAddr{Name:"",Net:"unixgram"})
 if runtime.GOOS=="darwin" {fmt.Println("empty address",errors.Is(err,syscall.EINVAL));return}
 if err!=nil{panic(err)};defer server.Close()
 addr:=server.LocalAddr().(*net.UnixAddr)
 if len(addr.Name)<=1||addr.Name[0]!='@'{panic("actual autobind address absent")}
 client,err:=net.DialUnix("unixgram",nil,addr);if err!=nil{panic(err)};defer client.Close()
 if client.RemoteAddr().String()!=addr.String(){panic("actual peer address differs")}
 if _,err=client.Write([]byte("actual autobind"));err!=nil{panic(err)}
 if err=server.SetReadDeadline(time.Now().Add(time.Second));err!=nil{panic(err)}
 b:=make([]byte,32);n,from,err:=server.ReadFromUnix(b)
 if err!=nil||from!=nil||string(b[:n])!="actual autobind"{panic("actual autobind packet differs")}
 fmt.Println("empty address",true)
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
