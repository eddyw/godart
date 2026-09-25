package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTCPSocketAddresses(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/tcpaddresses\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "encoding/json"
 "net"
 "os"
)

type observation struct {
 Network string
 Point string
 Length int
 IP string
 Is4 bool
 Is4In6 bool
 Is6 bool
}

func show(network, point string, a *net.TCPAddr) {
 ip:=a.AddrPort().Addr()
 r:=observation{network,point,len(a.IP),a.IP.String(),ip.Is4(),ip.Is4In6(),ip.Is6()}
 if err:=json.NewEncoder(os.Stdout).Encode(r);err!=nil {panic(err)}
}

func main() {
 for _, item:=range []struct{network,address string}{{"tcp","127.0.0.1:0"},{"tcp4","127.0.0.1:0"},{"tcp6","[::1]:0"}} {
  a,err:=net.ResolveTCPAddr(item.network,item.address);if err!=nil{panic(err)}
  show(item.network,"resolved",a)
  l,err:=net.ListenTCP(item.network,a);if err!=nil{panic(err)}
  show(item.network,"listener",l.Addr().(*net.TCPAddr))
  c,err:=net.Dial(item.network,l.Addr().String());if err!=nil{panic(err)}
  peer,err:=l.Accept();if err!=nil{panic(err)}
  show(item.network,"dial-local",c.LocalAddr().(*net.TCPAddr))
  show(item.network,"dial-remote",c.RemoteAddr().(*net.TCPAddr))
  show(item.network,"accept-local",peer.LocalAddr().(*net.TCPAddr))
  show(item.network,"accept-remote",peer.RemoteAddr().(*net.TCPAddr))
  c.Close();peer.Close();l.Close()
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
