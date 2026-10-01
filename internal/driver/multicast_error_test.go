package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestMulticastErrorMetadata(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native multicast errors require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/multicasterrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("encoding/json";"errors";"net";"os";"syscall")
func main(){
 type record struct { Success bool; Error string; Op string; Net string; SourceNil bool; SameAddr bool; Syscall string; Errno int }
 records:=[]record{}
 for _,address:=range []net.UDPAddr{
  {IP:net.IPv4(239,255,42,1)},
  {IP:net.IPv4(239,255,42,1),Port:-1},
  {IP:net.IPv4(239,255,42,1),Port:65536},
  {IP:net.IPv4(127,0,0,1)},
 }{
  conn,err:=net.ListenMulticastUDP("udp4",nil,&address)
  result:=record{Success:conn!=nil}
  if conn!=nil { if err:=conn.Close();err!=nil{panic(err)} }
  if err!=nil{result.Error=err.Error()}
  var op *net.OpError
  if errors.As(err,&op){result.Op=op.Op;result.Net=op.Net;result.SourceNil=op.Source==nil;result.SameAddr=op.Addr==&address}
  var call *os.SyscallError
  if errors.As(err,&call){result.Syscall=call.Syscall}
  var errno syscall.Errno
  if errors.As(err,&errno){result.Errno=int(errno)}
  records=append(records,result)
 }
 if err:=json.NewEncoder(os.Stdout).Encode(records);err!=nil{panic(err)}
}
`)
	native := filepath.Join(dir, "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := runProgram(t, dir, native)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, out, "dart", "bin/main.dart")
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("multicast error metadata differs: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}
