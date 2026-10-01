package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxMulticastInterfaceIndices(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux kernel multicast interface selection")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/multicastindices\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/interface_test.go"), []string{"routeStats", "validateInterfaceMulticastAddrs"})
	source = strings.Replace(source, "package net\n", "package main\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream.go"), source)
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("encoding/json";"errors";"net";"os";"sort";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 interfaces,err:=net.Interfaces();must(err)
 sort.Slice(interfaces,func(i,j int)bool{return interfaces[i].Index<interfaces[j].Index})
 var selected *net.Interface
 for i:=range interfaces{
  iface:=&interfaces[i]
  if iface.Flags&net.FlagUp==0||iface.Flags&net.FlagMulticast==0||iface.Flags&net.FlagLoopback!=0{continue}
  addresses,err:=iface.Addrs();must(err)
  for _,address:=range addresses{if address.(*net.IPNet).IP.To4()!=nil{selected=iface;break}}
  if selected!=nil{break}
 }
 if selected==nil{panic("no multicast interface with an IPv4 address")}
 type record struct{Requested int;Success bool;Members []int;Error string;Op string;Net string;SameAddr bool;Syscall string;Errno int}
 records:=[]record{}
 upper:=int64(1)<<32
 wrapped:=upper+int64(selected.Index)
 for i,index:=range []int{selected.Index,0,int(wrapped),int(upper),-1,1<<31-1}{
  group:=net.IPv4(239,255,78,byte(90+i))
  address:=&net.UDPAddr{IP:group}
  conn,err:=net.ListenMulticastUDP("udp4",&net.Interface{Index:index},address)
  result:=record{Requested:index,Success:conn!=nil,Members:[]int{}}
  if err!=nil{result.Error=err.Error()}
  var op *net.OpError
  if errors.As(err,&op){result.Op=op.Op;result.Net=op.Net;result.SameAddr=op.Addr==address}
  var call *os.SyscallError
  if errors.As(err,&call){result.Syscall=call.Syscall}
  var errno syscall.Errno
  if errors.As(err,&errno){result.Errno=int(errno)}
  if conn!=nil{
   for _,iface:=range interfaces{
    addresses,err:=iface.MulticastAddrs();must(err)
    _,err=validateInterfaceMulticastAddrs(addresses);must(err)
    for _,address:=range addresses{if address.(*net.IPAddr).IP.Equal(group){result.Members=append(result.Members,iface.Index)}}
   }
   if len(result.Members)==0{panic("joined multicast group is absent")}
   must(conn.Close())
  }
  records=append(records,result)
 }
 must(json.NewEncoder(os.Stdout).Encode(records))
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
				t.Fatalf("Linux multicast indexes differ: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}
