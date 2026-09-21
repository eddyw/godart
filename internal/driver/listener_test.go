package driver_test

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTCPListenerFamilies(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("listener family validation requires macOS or Linux")
	}
	probe, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback is unavailable: %v", err)
	}
	probe.Close()
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/listener\n\ngo 1.27\n\nrequire example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n")
	for _, item := range []struct {
		name  string
		decls []string
	}{
		{"listen_test.go", []string{"tcpListenerTests", "checkSecondListener"}},
		{"tcpsock_test.go", []string{"tcpListenerNameTests", "TestTCPListenerName"}},
	} {
		source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net", item.name), item.decls)
		source = strings.Replace(source, "package net\n", "package listener\nimport . \"net\"\n", 1)
		source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
		writeFile(t, filepath.Join(dir, item.name), source)
	}
	if err := os.Mkdir(filepath.Join(dir, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, "../overlay/std/internal/testenv/testenv.go", []string{"MustHaveExternalNetwork"})
	writeFile(t, filepath.Join(dir, "testenv/testenv.go"), strings.TrimPrefix(source, "//go:build dart\n"))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "testenv/LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "families_test.go"), `package listener
import ("context"; "errors"; "fmt"; . "net"; "strconv"; "syscall"; "testing"; "time")
func TestAddressFamilies(t *testing.T) {
 for _, tt := range tcpListenerTests {
  for _, api := range []string{"Listen", "ListenTCP", "ListenConfig"} {
   t.Run(api+"/"+tt.network+"/"+tt.address, func(t *testing.T) {
    address := JoinHostPort(tt.address,"0")
    var ln Listener
    var err error
    switch api {
    case "Listen": ln,err=Listen(tt.network,address)
    case "ListenTCP":
     var addr *TCPAddr
     addr,err=ResolveTCPAddr(tt.network,address)
     if err==nil { ln,err=ListenTCP(tt.network,addr) }
    case "ListenConfig":
     var lc ListenConfig
     ln,err=lc.Listen(context.Background(),tt.network,address)
    }
    if err!=nil {t.Fatal(err)}
    defer ln.Close()
    addr:=ln.Addr().(*TCPAddr)
    if addr.Port==0 || addr.Network()!="tcp" {t.Fatalf("invalid listener address %v",addr)}
    ip:=ParseIP(tt.address)
    want4:=tt.network=="tcp4" || (tt.network=="tcp" && ip.To4()!=nil && !ip.IsUnspecified())
    if got4:=addr.IP.To4()!=nil;got4!=want4 {t.Fatalf("listener family %v, IPv4=%v, want %v",addr,got4,want4)}
    second,err:=Listen(tt.network,JoinHostPort(tt.address,strconv.Itoa(addr.Port)))
    if err==nil {second.Close()}
    if err:=checkSecondListener(tt.network,tt.address,err);err!=nil {t.Fatal(err)}
    peer:=addr.IP.String()
    if addr.IP.IsUnspecified() { if want4 {peer="127.0.0.1"} else {peer="::1"} }
    conn,err:=DialTimeout("tcp",JoinHostPort(peer,strconv.Itoa(addr.Port)),time.Second)
    if err!=nil {t.Fatal(err)}
    defer conn.Close()
    ln.(*TCPListener).SetDeadline(time.Now().Add(time.Second))
    accepted,err:=ln.Accept()
    if err!=nil {t.Fatal(err)}
    accepted.Close()
   })
  }
 }
}
func TestLocalhost(t *testing.T) {
 for _,network:=range []string{"tcp","tcp4"} {
  ln,err:=Listen(network,"localhost:0");if err!=nil{t.Fatal(err)}
  addr:=ln.Addr().(*TCPAddr)
  if !addr.IP.IsLoopback()||addr.IP.To4()==nil {t.Errorf("%s localhost: %v",network,addr)}
  ln.Close()
 }
}
func TestSeparateFamilies(t *testing.T) {
 for _, first:=range []string{"tcp4","tcp6","tcp"} {
  t.Run(first,func(t *testing.T){
   ln,err:=Listen(first,":0");if err!=nil{t.Fatal(err)};defer ln.Close()
   second:="tcp4";if first=="tcp4"{second="tcp6"}
   next,err:=Listen(second,fmt.Sprintf(":%d",ln.Addr().(*TCPAddr).Port))
   if err==nil{next.Close()}
   if (err==nil)!=(first!="tcp"){t.Fatalf("%s then %s: %v",first,second,err)}
  })
 }
}
func TestAddressMismatch(t *testing.T) {
 for _,tt:=range []struct{network,address string}{
  {"tcp4","[::1]:0"},{"tcp6","127.0.0.1:0"},{"tcp6","[::ffff:127.0.0.1]:0"},
  {"tcp","127.0.0.1:-1"},{"tcp","127.0.0.1:65536"},
 } {
  ln,err:=Listen(tt.network,tt.address);if err==nil{ln.Close();t.Fatalf("accepted %v",tt)}
  var op *OpError
  var addr *AddrError
  if !errors.As(err,&op)||op.Op!="listen"||op.Net!=tt.network||!errors.As(err,&addr){t.Fatalf("incorrect error: %T %v",err,err)}
 }
 for _,tt:=range []struct{network string;addr *TCPAddr}{
  {"tcp4",&TCPAddr{IP:IPv6loopback}},
  {"tcp",&TCPAddr{IP:IPv4(127,0,0,1),Port:-1}},
  {"tcp",&TCPAddr{IP:IPv4(127,0,0,1),Port:65536}},
 } {
  ln,err:=ListenTCP(tt.network,tt.addr);if err==nil{ln.Close();t.Fatalf("accepted typed address %v",tt)}
  var op *OpError
  var addr *AddrError
  if !errors.As(err,&op)||op.Op!="listen"||op.Net!=tt.network {t.Fatalf("incorrect typed error: %T %v",err,err)}
  if tt.addr.Port==0 {if !errors.As(err,&addr){t.Fatalf("incorrect address error: %v",err)}} else if !errors.Is(err,syscall.EINVAL){t.Fatalf("incorrect port error: %v",err)}
 }
 for _,network:=range []string{"tcp4","tcp6"} {
  ln,err:=ListenTCP(network,nil);if err!=nil{t.Fatal(err)}
  if got4:=ln.Addr().(*TCPAddr).IP.To4()!=nil;got4!=(network=="tcp4"){t.Fatalf("nil address: %v",ln.Addr())}
  ln.Close()
 }
}
`)
	run(t, dir, "go", "test", "-v", ".")
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v")
	executable := filepath.Join(out, "listeners")
	run(t, out, "dart", "compile", "exe", stats.Binaries[0].Entry, "-o", executable)
	run(t, dir, executable, "-test.v")
}

func TestTCPListenerWildcard(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/wildcard\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "net")
func show(ln *net.TCPListener, err error) {
 if err!=nil {fmt.Println("error");return}
 fmt.Println(ln.Addr().(*net.TCPAddr).IP)
 ln.Close()
}
func main() {
 for _,address:=range []string{"",":0","0.0.0.0:0","[::]:0"} {
  ln,err:=net.Listen("tcp",address)
  if err!=nil {fmt.Println("error");continue}
  show(ln.(*net.TCPListener),nil)
 }
 for _,ip:=range []net.IP{nil,net.IPv4zero,net.IPv6zero} {
  show(net.ListenTCP("tcp",&net.TCPAddr{IP:ip}))
 }
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	got := runProgram(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
	if got != want {
		t.Fatalf("wildcard addresses differ:\nGo:\n%s\nGart:\n%s", want, got)
	}
	executable := filepath.Join(out, "wildcards")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", executable)
	if got := runProgram(t, dir, executable); got != want {
		t.Fatalf("AOT wildcard addresses differ:\nGo:\n%s\nGart:\n%s", want, got)
	}
}
