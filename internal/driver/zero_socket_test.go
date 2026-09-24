package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestSocketZeroValues(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/zerosockets\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "fixture_test.go"), `package zerosockets
import("bytes";"errors";"io";"net";"net/netip";"syscall";"testing";"time")
func TestZeroSockets(t *testing.T) {
 listener,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 live,err:=net.Dial("tcp",listener.Addr().String());if err!=nil{t.Fatal(err)}
 defer live.Close()
 peer,err:=listener.Accept();if err!=nil{t.Fatal(err)}
 defer peer.Close()
 listener.Close()
 if err:=live.SetDeadline(time.Now().Add(time.Second));err!=nil{t.Fatal(err)}
 var tcp net.TCPConn
 var unix net.UnixConn
 var udp net.UDPConn
 deadline:=time.Unix(1,0)
 for _,c:=range []struct{name string;c net.Conn}{{"tcp",&tcp},{"unix",&unix},{"udp",&udp}} {
  t.Run(c.name,func(t *testing.T){
   for _,op:=range []struct{name string;call func()error}{
    {"read",func()error{n,e:=c.c.Read(nil);if n!=0{t.Errorf("read count %d",n)};return e}},
    {"write",func()error{n,e:=c.c.Write(nil);if n!=0{t.Errorf("write count %d",n)};return e}},
    {"deadline",func()error{return c.c.SetDeadline(deadline)}},
    {"read-deadline",func()error{return c.c.SetReadDeadline(deadline)}},
    {"write-deadline",func()error{return c.c.SetWriteDeadline(deadline)}},
   } {t.Run(op.name,func(t *testing.T){expectInvalid(t,op.call)})}
   t.Run("local-address",func(t *testing.T){defer reportPanic(t);if a:=c.c.LocalAddr();a!=nil{t.Errorf("address %v, want nil",a)}})
   t.Run("remote-address",func(t *testing.T){defer reportPanic(t);if a:=c.c.RemoteAddr();a!=nil{t.Errorf("address %v, want nil",a)}})
  })
 }
 for _,op:=range []struct{name string;call func()error}{
  {"tcp-no-delay",func()error{return tcp.SetNoDelay(true)}},
  {"tcp-keepalive",func()error{return tcp.SetKeepAlive(true)}},
  {"tcp-keepalive-period",func()error{return tcp.SetKeepAlivePeriod(-1)}},
  {"tcp-keepalive-config",func()error{return tcp.SetKeepAliveConfig(net.KeepAliveConfig{})}},
  {"tcp-read-buffer",func()error{return tcp.SetReadBuffer(4096)}},
  {"tcp-write-buffer",func()error{return tcp.SetWriteBuffer(4096)}},
  {"unix-read-buffer",func()error{return unix.SetReadBuffer(4096)}},
  {"unix-write-buffer",func()error{return unix.SetWriteBuffer(4096)}},
  {"udp-read-buffer",func()error{return udp.SetReadBuffer(4096)}},
  {"udp-write-buffer",func()error{return udp.SetWriteBuffer(4096)}},
  {"tcp-read-from",func()error{n,e:=tcp.ReadFrom(bytes.NewReader(nil));if n!=0{t.Errorf("count %d",n)};return e}},
  {"tcp-write-to",func()error{n,e:=tcp.WriteTo(io.Discard);if n!=0{t.Errorf("count %d",n)};return e}},
  {"unix-read-from",func()error{n,a,e:=unix.ReadFrom(nil);if n!=0||a!=nil{t.Errorf("result %d %v",n,a)};return e}},
  {"unix-read-from-unix",func()error{n,a,e:=unix.ReadFromUnix(nil);if n!=0||a!=nil{t.Errorf("result %d %v",n,a)};return e}},
  {"udp-read-from",func()error{n,a,e:=udp.ReadFrom(nil);if n!=0||a!=nil{t.Errorf("result %d %v",n,a)};return e}},
  {"udp-read-from-udp",func()error{n,a,e:=udp.ReadFromUDP(nil);if n!=0||a!=nil{t.Errorf("result %d %v",n,a)};return e}},
  {"udp-read-from-addrport",func()error{n,a,e:=udp.ReadFromUDPAddrPort(nil);if n!=0||a.IsValid(){t.Errorf("result %d %v",n,a)};return e}},
  {"udp-write-to",func()error{n,e:=udp.WriteTo(nil,nil);if n!=0{t.Errorf("count %d",n)};return e}},
  {"udp-write-to-udp",func()error{n,e:=udp.WriteToUDP(nil,nil);if n!=0{t.Errorf("count %d",n)};return e}},
  {"udp-write-to-addrport",func()error{n,e:=udp.WriteToUDPAddrPort(nil,netip.AddrPort{});if n!=0{t.Errorf("count %d",n)};return e}},
  {"tcp-accept",func()error{v,e:=new(net.TCPListener).Accept();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
  {"tcp-accept-tcp",func()error{v,e:=new(net.TCPListener).AcceptTCP();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
  {"unix-accept",func()error{v,e:=new(net.UnixListener).Accept();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
  {"unix-accept-unix",func()error{v,e:=new(net.UnixListener).AcceptUnix();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
  {"nil-tcp-accept",func()error{v,e:=(*net.TCPListener)(nil).Accept();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
  {"nil-tcp-accept-tcp",func()error{v,e:=(*net.TCPListener)(nil).AcceptTCP();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
  {"nil-unix-accept",func()error{v,e:=(*net.UnixListener)(nil).Accept();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
  {"nil-unix-accept-unix",func()error{v,e:=(*net.UnixListener)(nil).AcceptUnix();if v!=nil{v.Close();t.Error("accepted connection")};return e}},
 } {t.Run(op.name,func(t *testing.T){expectInvalid(t,op.call)})}
 t.Run("live-ownership",func(t *testing.T){
  if _,err:=live.Write([]byte("live"));err!=nil{t.Fatal(err)}
  peer.SetReadDeadline(time.Now().Add(time.Second))
  b:=make([]byte,4);if _,err:=io.ReadFull(peer,b);err!=nil{t.Fatal(err)}
  if string(b)!="live"{t.Fatalf("payload %q",b)}
 })
}
func reportPanic(t *testing.T){t.Helper();if p:=recover();p!=nil{t.Errorf("panic: %v",p)}}
func expectInvalid(t *testing.T,call func()error){t.Helper();defer reportPanic(t);if err:=call();!errors.Is(err,syscall.EINVAL){t.Errorf("error %v, want EINVAL",err)}}
`)
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var r programResult
			if target == "vm" {
				r = programOutcome(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v")
			} else {
				r = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v"})
			}
			t.Log(r.Text)
			if r.Exit != 0 {
				t.Fatal(r.Text)
			}
		})
	}
}
