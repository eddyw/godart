package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnixgramDialWildcardContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix dial addresses require native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unixdialwildcard\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"net";"os";"path/filepath";"reflect";"runtime";"syscall";"errors";"time")
func opAddr(a *net.UnixAddr) net.Addr {if a==nil{return nil};return a}
func main(){
 dir,err:=os.MkdirTemp("","udw");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 serverAddr:=&net.UnixAddr{Name:filepath.Join(dir,"s"),Net:"unixgram"};server,err:=net.ListenUnixgram("unixgram",serverAddr);if err!=nil{panic(err)};defer server.Close()
 for i,c:=range [][2]int{{0,0},{1,0},{0,1},{1,1},{1,2},{2,1},{2,0},{0,2}} {
  var local,remote *net.UnixAddr
  if c[0]==1 {local=&net.UnixAddr{Net:"unixgram"}} else if c[0]==2 {local=&net.UnixAddr{Name:filepath.Join(dir,fmt.Sprint("l",i)),Net:"unixgram"}}
  if c[1]==1 {remote=&net.UnixAddr{Net:"unixgram"}} else if c[1]==2 {remote=serverAddr}
  client,err:=net.DialUnix("unixgram",local,remote)
  if err!=nil{op:=err.(*net.OpError);fmt.Println(i,"error",op.Err.Error(),errors.Is(err,syscall.EINVAL),reflect.DeepEqual(op.Source,opAddr(local)),reflect.DeepEqual(op.Addr,opAddr(remote)));continue}
  expectedLocal:="";if runtime.GOOS=="linux"{expectedLocal="@"};if c[0]==2{expectedLocal=local.Name}
  peerOK:=client.RemoteAddr()==nil;if c[1]==2{peerOK=reflect.DeepEqual(client.RemoteAddr(),serverAddr)}
  fmt.Println(i,"addresses",client.LocalAddr().String()==expectedLocal,peerOK)
  payload:=[]byte{0,255,128,byte(i)};var n int
  if c[1]==2{n,err=client.Write(payload)}else{n,err=client.WriteToUnix(payload,serverAddr)}
  if err!=nil||n!=len(payload){panic("dial packet write failed")}
  if err=server.SetReadDeadline(time.Now().Add(time.Second));err!=nil{panic(err)}
  b:=make([]byte,16);n,from,err:=server.ReadFromUnix(b);if err!=nil||string(b[:n])!=string(payload){panic("dial packet differs")}
  fromOK:=from==nil;if c[0]==2{fromOK=reflect.DeepEqual(from,local)};fmt.Println(i,"packet source",fromOK,"ownership",client.Close()==nil)
 }
 missing:=&net.UnixAddr{Name:filepath.Join(dir,"absent"),Net:"unixgram"};empty:=&net.UnixAddr{Net:"unixgram"}
 _,err=net.DialUnix("unixgram",empty,missing);if err==nil{panic("absent destination accepted")};op:=err.(*net.OpError)
 fmt.Println("absent destination",errors.Is(err,syscall.ENOENT),reflect.DeepEqual(op.Source,opAddr(empty)),reflect.DeepEqual(op.Addr,opAddr(missing)))
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
