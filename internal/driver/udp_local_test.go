package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUDPLocalAddress(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("connected UDP address validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udplocal\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/udpsock_test.go"), []string{"TestUDPConnLocalAndRemoteNames"})
	source = strings.Replace(source, "package net\n", "package udplocal\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "local_test.go"), `package udplocal
import (. "net"; "testing"; "time")
func testableNetwork(network string) bool {
 c,err:=ListenUDP(network,&UDPAddr{IP:IPv4(127,0,0,1)})
 if err!=nil{return false};c.Close();return true
}
func TestLocalBind(t *testing.T) {
 for _,host:=range []string{"127.0.0.1","::1"} {
  for _,mode:=range []string{"automatic","explicit","port","occupied"} {
   t.Run(host+"/"+mode,func(t *testing.T){
    ip:=ParseIP(host)
    server,err:=ListenUDP("udp",&UDPAddr{IP:ip})
    if err!=nil{t.Fatal(err)}
    defer server.Close()
    var local *UDPAddr
    if mode!="automatic" {local=&UDPAddr{IP:ip}}
    if mode=="port" {
     reserve,err:=ListenUDP("udp",local);if err!=nil{t.Fatal(err)}
     local=reserve.LocalAddr().(*UDPAddr);reserve.Close()
    }
    if mode=="occupied"{local=server.LocalAddr().(*UDPAddr)}
    client,err:=DialUDP("udp",local,server.LocalAddr().(*UDPAddr))
    if mode=="occupied"{if err==nil{client.Close();t.Fatal("accepted occupied local address")};return}
    if err!=nil{t.Fatal(err)}
    defer client.Close()
    actual:=client.LocalAddr().(*UDPAddr)
    if !actual.IP.Equal(ip)||actual.Port==0 {t.Fatalf("local address %v",actual)}
    if mode=="port"&&actual.Port!=local.Port {t.Fatalf("local port %d, want %d",actual.Port,local.Port)}
    client.SetWriteDeadline(time.Now().Add(time.Second))
    if _,err:=client.Write([]byte("hello"));err!=nil{t.Fatal(err)}
    server.SetReadDeadline(time.Now().Add(time.Second))
    buf:=make([]byte,10);n,from,err:=server.ReadFromUDP(buf)
    if err!=nil{t.Fatal(err)}
    if string(buf[:n])!="hello"||!from.IP.Equal(actual.IP)||from.Port!=actual.Port {t.Fatalf("source %v, local %v, payload %q",from,actual,buf[:n])}
   })
  }
 }
}
func TestCloseRebind(t *testing.T) {
 for i:=0;i<20;i++ {
  server,err:=ListenUDP("udp",&UDPAddr{IP:IPv4(127,0,0,1)})
  if err!=nil{t.Fatal(err)}
  reserve,err:=ListenUDP("udp",&UDPAddr{IP:IPv4(127,0,0,1)})
  if err!=nil{server.Close();t.Fatal(err)}
  address:=reserve.LocalAddr().(*UDPAddr)
  if err:=reserve.Close();err!=nil{server.Close();t.Fatal(err)}
  client,err:=DialUDP("udp",address,server.LocalAddr().(*UDPAddr))
  if err!=nil{server.Close();t.Fatal(err)}
  client.Close();server.Close()
 }
}
`)
	run(t, dir, "go", "test", "-v", ".")
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			args := []string{"-test.v", "-test.run=^TestUDPConnLocalAndRemoteNames$"}
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "udp-local", "net", "^TestUDPConnLocalAndRemoteNames$", target, filepath.Join(runtime.GOROOT(), "src/net"), result.Text, result.Exit)
			if target == "vm" {
				run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.run=^Test(LocalBind|CloseRebind)$")
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v", "-test.run=^Test(LocalBind|CloseRebind)$"})
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
