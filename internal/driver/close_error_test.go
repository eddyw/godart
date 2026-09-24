package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSocketCloseErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("this adapter covers Linux and macOS socket errors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/closeerrors\n\ngo 1.27\n")
	root := filepath.Join(runtime.GOROOT(), "src/net")
	for _, item := range []struct {
		file  string
		names []string
	}{
		{"error_test.go", []string{"OpError.isValid", "parseCloseError", "TestCloseError"}},
		{"error_unix_test.go", []string{"isPlatformError"}},
		{"pipe.go", []string{"pipeAddr"}},
		{"file.go", []string{"fileAddr"}},
		{"mockserver_test.go", []string{"testUnixAddr"}},
	} {
		source := selectedUpstreamSource(t, filepath.Join(root, item.file), item.names)
		if item.file == "error_test.go" {
			source = strings.Replace(source, "package net\n", "package closeerrors\nimport . \"net\"\n", 1)
			source = strings.Replace(source, "func (e *OpError) isValid() error", "func validOpError(e *OpError) error", 1)
			source = strings.ReplaceAll(source, "err.isValid()", "validOpError(err)")
		} else {
			source = strings.Replace(source, "package net\n", "package closeerrors\n", 1)
		}
		writeFile(t, filepath.Join(dir, item.file), source)
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "fixture_test.go"), `package closeerrors
import ("context"; "errors"; "io"; "net"; "net/netip"; "syscall"; "testing")
func newLocalListener(t *testing.T, network string) net.Listener {
 t.Helper()
 l,err:=net.Listen(network,"127.0.0.1:0")
 if err!=nil {t.Fatal(err)}
 return l
}
func testableNetwork(network string) bool {return network=="udp"}
func TestZeroClose(t *testing.T) {
 l:=newLocalListener(t,"tcp")
 defer l.Close()
 c,err:=net.Dial("tcp",l.Addr().String())
 if err!=nil {t.Fatal(err)}
 defer c.Close()
 for _,v:=range []io.Closer{new(net.TCPConn),new(net.UDPConn),new(net.UnixConn),new(net.TCPListener),new(net.UnixListener),(*net.TCPListener)(nil),(*net.UnixListener)(nil)} {
  if err:=v.Close();!errors.Is(err,syscall.EINVAL) {t.Fatalf("%T.Close: %v, want EINVAL",v,err)}
 }
 for _,close:=range []func()error{new(net.TCPConn).CloseRead,new(net.TCPConn).CloseWrite,new(net.UnixConn).CloseRead,new(net.UnixConn).CloseWrite} {
  if err:=close();!errors.Is(err,syscall.EINVAL) {t.Fatalf("zero half-close: %v, want EINVAL",err)}
 }
 if _,err:=c.Write([]byte("still open"));err!=nil {t.Fatal(err)}
 next,err:=net.Dial("tcp",l.Addr().String())
 if err!=nil {t.Fatal(err)}
 next.Close()
}
func sameAddress(a,b net.Addr) bool {
 if a==nil||b==nil {return a==b}
 return a.Network()==b.Network()&&a.String()==b.String()
}
func closeMetadata(t *testing.T,c io.Closer,network string,source,addr net.Addr) {
 t.Helper()
 if err:=c.Close();err!=nil {t.Fatal(err)}
 err:=c.Close()
 op,ok:=err.(*net.OpError)
 if !ok {t.Fatalf("%T.Close: %T, want *net.OpError",c,err)}
 if op.Op!="close"||op.Net!=network||!sameAddress(op.Source,source)||!sameAddress(op.Addr,addr)||!errors.Is(op.Err,net.ErrClosed) {
  t.Fatalf("%T.Close: %+v, want network=%s source=%v addr=%v and ErrClosed",c,op,network,source,addr)
 }
}
func TestCloseMetadata(t *testing.T) {
 t.Run("tcp4",func(t *testing.T){
  l,err:=net.ListenTCP("tcp4",&net.TCPAddr{IP:net.IPv4(127,0,0,1)})
  if err!=nil {t.Fatal(err)}
  defer l.Close()
  c,err:=net.DialTCP("tcp4",nil,l.Addr().(*net.TCPAddr))
  if err!=nil {t.Fatal(err)}
  peer,err:=l.AcceptTCP();if err!=nil {t.Fatal(err)}
  closeMetadata(t,c,"tcp4",c.LocalAddr(),c.RemoteAddr())
  closeMetadata(t,peer,"tcp4",peer.LocalAddr(),peer.RemoteAddr())
  d:=net.Dialer{LocalAddr:&net.TCPAddr{Port:-1}}
  c,err=d.DialTCP(context.Background(),"tcp4",netip.AddrPort{},l.Addr().(*net.TCPAddr).AddrPort())
  if err!=nil {t.Fatal(err)}
  peer,err=l.AcceptTCP();if err!=nil {t.Fatal(err)}
  closeMetadata(t,c,"tcp4",c.LocalAddr(),c.RemoteAddr())
  peer.Close()
  closeMetadata(t,l,"tcp4",nil,l.Addr())
 })
 t.Run("udp4",func(t *testing.T){
  l,err:=net.ListenUDP("udp4",&net.UDPAddr{IP:net.IPv4(127,0,0,1)})
  if err!=nil {t.Fatal(err)}
  defer l.Close()
  c,err:=net.DialUDP("udp4",nil,l.LocalAddr().(*net.UDPAddr))
  if err!=nil {t.Fatal(err)}
  closeMetadata(t,c,"udp4",c.LocalAddr(),c.RemoteAddr())
  d:=net.Dialer{LocalAddr:&net.TCPAddr{Port:-1}}
  c,err=d.DialUDP(context.Background(),"udp4",netip.AddrPort{},l.LocalAddr().(*net.UDPAddr).AddrPort())
  if err!=nil {t.Fatal(err)}
  closeMetadata(t,c,"udp4",c.LocalAddr(),c.RemoteAddr())
  closeMetadata(t,l,"udp4",l.LocalAddr(),nil)
 })
 t.Run("unix",func(t *testing.T){
  l,err:=net.ListenUnix("unix",&net.UnixAddr{Name:testUnixAddr(t),Net:"unix"})
  if err!=nil {t.Fatal(err)}
  defer l.Close()
  c,err:=net.DialUnix("unix",nil,l.Addr().(*net.UnixAddr));if err!=nil {t.Fatal(err)}
  peer,err:=l.AcceptUnix();if err!=nil {t.Fatal(err)}
  closeMetadata(t,c,"unix",c.LocalAddr(),c.RemoteAddr())
  closeMetadata(t,peer,"unix",peer.LocalAddr(),peer.RemoteAddr())
  d:=net.Dialer{LocalAddr:&net.TCPAddr{Port:-1}}
  c,err=d.DialUnix(context.Background(),"unix",nil,l.Addr().(*net.UnixAddr));if err!=nil {t.Fatal(err)}
  peer,err=l.AcceptUnix();if err!=nil {t.Fatal(err)}
  closeMetadata(t,c,"unix",c.LocalAddr(),c.RemoteAddr())
  peer.Close()
  closeMetadata(t,l,"unix",nil,l.Addr())
 })
}

`)
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			if target == "vm" {
				t.Log(run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v"))
			} else {
				result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v"})
				t.Log(result.Text)
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
