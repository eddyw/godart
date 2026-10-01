package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestIPv6Multicast(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native IPv6 multicast requires Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/ipv6multicast\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/interface_test.go"), []string{"routeStats", "validateInterfaceMulticastAddrs"})
	source = strings.Replace(source, "package net\n", "package main\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream.go"), source)
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("encoding/json";"errors";"fmt";"net";"os";"sort";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 interfaces,err:=net.Interfaces();must(err)
 sort.Slice(interfaces,func(i,j int)bool{return interfaces[i].Index<interfaces[j].Index})
 var loopback *net.Interface
 for i:=range interfaces{if interfaces[i].Flags&net.FlagLoopback!=0{loopback=&interfaces[i];break}}
 if loopback==nil{panic("no loopback interface")}
 upper:=int64(1)<<32
 wrapped:=upper+int64(loopback.Index)
 cases:=[]struct{Network string;Interface *net.Interface}{
  {"udp",loopback},{"udp",&net.Interface{Index:int(wrapped)}},{"udp",nil},{"udp",&net.Interface{}},{"udp",&net.Interface{Index:-1}},{"udp4",loopback},{"udp4",nil},
 }
 type record struct{Case int;Success bool;LocalIP []byte;LocalZone string;HasPort bool;Members []int;Error string;Op string;Net string;SameAddr bool;Syscall string;Errno int}
 records:=[]record{}
 for i,c:=range cases{
  group:=net.ParseIP(fmt.Sprintf("ff02::%x",0x114+i))
  address:=&net.UDPAddr{IP:group}
  conn,err:=net.ListenMulticastUDP(c.Network,c.Interface,address)
  result:=record{Case:i,Success:conn!=nil,Members:[]int{}}
  if err!=nil{result.Error=err.Error()}
  var op *net.OpError
  if errors.As(err,&op){result.Op=op.Op;result.Net=op.Net;result.SameAddr=op.Addr==address}
  var call *os.SyscallError
  if errors.As(err,&call){result.Syscall=call.Syscall}
  var errno syscall.Errno
  if errors.As(err,&errno){result.Errno=int(errno)}
  if conn!=nil{
   local:=conn.LocalAddr().(*net.UDPAddr);result.LocalIP=local.IP;result.LocalZone=local.Zone;result.HasPort=local.Port>0
   for _,iface:=range interfaces{
    addresses,err:=iface.MulticastAddrs();must(err)
    _,err=validateInterfaceMulticastAddrs(addresses);must(err)
    for _,address:=range addresses{if address.(*net.IPAddr).IP.Equal(group){result.Members=append(result.Members,iface.Index)}}
   }
   if len(result.Members)==0{panic("joined IPv6 multicast group is absent")}
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
				t.Fatalf("IPv6 multicast differs: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}
