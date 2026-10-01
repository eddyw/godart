package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDarwinMulticastInterfaceErrors(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS multicast interface errors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/darwinmulticastinterfaces\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("encoding/json";"errors";"net";"os";"syscall")
func main(){
 interfaces,err:=net.Interfaces();if err!=nil{panic(err)}
 var loopback *net.Interface
 for _,iface:=range interfaces{if iface.Flags&net.FlagLoopback!=0{loopback=&iface;break}}
 if loopback==nil{panic("no loopback interface")}
 type record struct{Index int;HasAddresses bool;AddressError string;Success bool;Error string;Op string;Net string;SourceNil bool;SameAddr bool;SameCause bool;Syscall string;Errno int}
 _,referenceErr:=net.InterfaceByName("absent-multicast-interface")
 var reference *net.OpError
 if !errors.As(referenceErr,&reference){panic("missing interface error has no OpError")}
 records:=[]record{}
 upper:=int64(1)<<32
 wrapped:=upper+int64(loopback.Index)
 for i,index:=range []int{loopback.Index,0,-1,1<<31-1,int(upper),int(wrapped)}{
  iface:=&net.Interface{Index:index}
  addresses,err:=iface.Addrs()
  result:=record{Index:index,HasAddresses:len(addresses)!=0}
  if err!=nil{result.AddressError=err.Error()}
  address:=&net.UDPAddr{IP:net.IPv4(239,255,79,byte(90+i))}
  conn,err:=net.ListenMulticastUDP("udp4",iface,address)
  result.Success=conn!=nil
  if conn!=nil{if err:=conn.Close();err!=nil{panic(err)}}
  if err!=nil{result.Error=err.Error()}
  result.SameCause=errors.Is(err,reference.Err)
  var op *net.OpError
  if errors.As(err,&op){result.Op=op.Op;result.Net=op.Net;result.SourceNil=op.Source==nil;result.SameAddr=op.Addr==address}
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
				t.Fatalf("macOS multicast interface errors differ: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}
