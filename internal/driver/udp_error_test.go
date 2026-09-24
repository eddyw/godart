package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUDPErrorCodes(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native datagram errno validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udperrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "fixture_test.go"), `package udperrors
import("errors";"net";"os";"syscall";"testing";"time")
func TestUDPErrorCodes(t *testing.T){
 for _,address:=range []string{"127.0.0.1:0","[::1]:0"}{t.Run(address,func(t *testing.T){
 p,e:=net.ListenPacket("udp",address);if e!=nil{t.Fatal(e)};defer p.Close()
 c,e:=net.Dial("udp",p.LocalAddr().String());if e!=nil{t.Fatal(e)};defer c.Close()
 for _,tc:=range []struct{name,call string;write func([]byte)(int,error);read func([]byte)(int,error)}{
 {"connected","write",c.Write,func(b []byte)(int,error){n,_,e:=p.ReadFrom(b);return n,e}},
 {"unconnected","sendto",func(b []byte)(int,error){return p.WriteTo(b,c.LocalAddr())},c.Read},
 }{t.Run(tc.name,func(t *testing.T){
 n,e:=tc.write(make([]byte,65536))
 if n!=0||!errors.Is(e,syscall.EMSGSIZE){t.Fatalf("write = %d %T %v, want 0 and EMSGSIZE",n,e,e)}
 var op *net.OpError;if !errors.As(e,&op)||op.Op!="write"||op.Net!="udp"||op.Source==nil||op.Addr==nil{t.Fatalf("operation error %#v",op)}
 var call *os.SyscallError;if !errors.As(e,&call)||call.Syscall!=tc.call{t.Fatalf("system call error %#v, want %s",call,tc.call)}
 p.SetReadDeadline(time.Now().Add(time.Second));c.SetReadDeadline(time.Now().Add(time.Second))
 payload:=[]byte("ok")
 if n,e:=tc.write(payload);n!=len(payload)||e!=nil{t.Fatalf("write after error = %d %v",n,e)}
 b:=make([]byte,16);n,e=tc.read(b)
 if n!=len(payload)||e!=nil||string(b[:n])!=string(payload){t.Fatalf("read after error = %d %v %q",n,e,b[:n])}

 })}
 })}
}

`)
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/udpsock_test.go"), []string{"TestUDPZeroByteBuffer", "TestUDPReadSizeError"})
	source = strings.Replace(source, "package net\n", "package udperrors\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "helpers_test.go"), `package udperrors
 import("net";"testing")
 func testableNetwork(network string)bool{return network=="udp"}
 func newLocalPacketListener(t testing.TB, network string)net.PacketConn{
  t.Helper();p,e:=net.ListenPacket(network,"127.0.0.1:0");if e!=nil{t.Fatal(e)};return p
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
			const filter = "^(TestUDPZeroByteBuffer|TestUDPReadSizeError)$"
			args := []string{"-test.v", "-test.run=" + filter}
			var r programResult
			if target == "vm" {
				r = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				r = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "udp-datagram-errors", "net", filter, target, filepath.Join(runtime.GOROOT(), "src/net"), r.Text, r.Exit)
			args = []string{"-test.v", "-test.run=^TestUDPErrorCodes$"}
			if target == "vm" {
				r = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				r = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			t.Log(r.Text)
			if r.Exit != 0 {
				t.Fatal(r.Text)
			}

		})
	}
}
