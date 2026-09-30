package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUDPSourceAddressFamilies(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udpsource\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"net";"time")
func main(){
 for _,tt:=range []struct{name,network,local,peer string}{
  {"ipv4","udp4","127.0.0.1","127.0.0.1"},
  {"ipv6","udp6","::1","::1"},
  {"mapped","udp","::","127.0.0.1"},
 } {
  for _,method:=range []string{"addrport","udpaddr","addr"} {
   for _,payload:=range [][]byte{nil,[]byte("message")} {
    server,err:=net.ListenUDP(tt.network,&net.UDPAddr{IP:net.ParseIP(tt.local)})
    if err!=nil{panic(err)}
    target:=&net.UDPAddr{IP:net.ParseIP(tt.peer),Port:server.LocalAddr().(*net.UDPAddr).Port}
    client,err:=net.DialUDP("udp",nil,target)
    if err!=nil{panic(err)}
    if _,err=client.Write(payload);err!=nil{panic(err)}
    if err=server.SetReadDeadline(time.Now().Add(2*time.Second));err!=nil{panic(err)}
    data:=make([]byte,32)
    var n int
    var address string
    var iplen int
    var port int
    switch method {
    case "addrport":
     n0,from,err:=server.ReadFromUDPAddrPort(data);if err!=nil{panic(err)}
     n=n0;address=from.Addr().String();iplen=from.Addr().BitLen()/8;port=int(from.Port())
    case "udpaddr":
     n0,from,err:=server.ReadFromUDP(data);if err!=nil{panic(err)}
     n=n0;address=from.IP.String();iplen=len(from.IP);port=from.Port
    case "addr":
     n0,from,err:=server.ReadFrom(data);if err!=nil{panic(err)}
     n=n0;u:=from.(*net.UDPAddr);address=u.IP.String();iplen=len(u.IP);port=u.Port
    }
    fmt.Printf("%s/%s/%d source=%s bytes=%d port=%t payload=%t\n",tt.name,method,len(payload),address,iplen,port==client.LocalAddr().(*net.UDPAddr).Port,n==len(payload)&&string(data[:n])==string(payload))
    client.Close();server.Close()
   }
  }
 }
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
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, result.Exit, result.Text)
			}
		})
	}
}
