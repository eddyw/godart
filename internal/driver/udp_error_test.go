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
func TestUDPReadError(t *testing.T){
 for _,tc:=range []struct{name,call string}{ {"read","read"},{"readfrom","recvfrom"} }{t.Run(tc.name,func(t *testing.T){
  p,e:=net.ListenPacket("udp","127.0.0.1:0");if e!=nil{t.Fatal(e)}
  c,e:=net.Dial("udp",p.LocalAddr().String());if e!=nil{p.Close();t.Fatal(e)};defer c.Close()
  if e:=p.Close();e!=nil{t.Fatal(e)}
  c.SetReadDeadline(time.Now().Add(2*time.Second))
  if _,e:=c.Write([]byte("probe"));e!=nil{t.Fatal(e)}
  b:=make([]byte,16);var n int
  if tc.name=="read"{n,e=c.Read(b)}else{n,_,e=c.(*net.UDPConn).ReadFromUDP(b)}
  if n!=0||!errors.Is(e,syscall.ECONNREFUSED){t.Fatalf("read = %d %T %v, want 0 and ECONNREFUSED",n,e,e)}
  var call *os.SyscallError;if !errors.As(e,&call)||call.Syscall!=tc.call{t.Fatalf("system call error %#v, want %s",call,tc.call)}
 })}
}

func TestUDPQueuedDatagrams(t *testing.T){
 for _,address:=range []string{"127.0.0.1:0","[::1]:0"}{t.Run(address,func(t *testing.T){
  p,e:=net.ListenPacket("udp",address);if e!=nil{t.Fatal(e)};defer p.Close()
  c,e:=net.Dial("udp",p.LocalAddr().String());if e!=nil{t.Fatal(e)};defer c.Close()
  payloads:=[][]byte{nil,[]byte("a"),nil,[]byte("b"),nil}
  for _,b:=range payloads{if n,e:=c.Write(b);n!=len(b)||e!=nil{t.Fatalf("write %d %v",n,e)}}
  p.SetReadDeadline(time.Now().Add(time.Second))
  for _,want:=range payloads{b:=make([]byte,16);n,_,e:=p.ReadFrom(b);if e!=nil||string(b[:n])!=string(want){t.Fatalf("read %q %v, want %q",b[:n],e,want)}}
 })}
}
func TestUDPConcurrentReads(t *testing.T){
 p,e:=net.ListenPacket("udp","127.0.0.1:0");if e!=nil{t.Fatal(e)};defer p.Close()
 c,e:=net.Dial("udp",p.LocalAddr().String());if e!=nil{t.Fatal(e)};defer c.Close()
 p.SetReadDeadline(time.Now().Add(2*time.Second))
 type result struct{data string;err error};done:=make(chan result,2);ready:=make(chan bool)
 for i:=0;i<2;i++{go func(){ready<-true;b:=make([]byte,16);n,_,e:=p.ReadFrom(b);done<-result{string(b[:n]),e}}()}
 <-ready;<-ready;time.Sleep(10*time.Millisecond)
 for _,message:=range []string{"a","b"}{
  if _,e:=c.Write([]byte(message));e!=nil{t.Fatal(e)}
  r:=<-done;if r.data!=message||r.err!=nil{t.Fatalf("read = %q %v, want %q",r.data,r.err,message)}
 }
}

func TestUDPBlockedReads(t *testing.T){
 for _,action:=range []string{"close","deadline"}{t.Run(action,func(t *testing.T){
  p,e:=net.ListenPacket("udp","127.0.0.1:0");if e!=nil{t.Fatal(e)};defer p.Close()
  ready:=make(chan bool);done:=make(chan error,3)
  for i:=0;i<3;i++{go func(){ready<-true;_,_,e:=p.ReadFrom(make([]byte,16));done<-e}()}
  <-ready;<-ready;<-ready;time.Sleep(10*time.Millisecond)
  want:=net.ErrClosed
  if action=="close"{if e:=p.Close();e!=nil{t.Fatal(e)}}else{want=os.ErrDeadlineExceeded;p.SetReadDeadline(time.Unix(1,0))}
  for i:=0;i<3;i++{if e:=<-done;!errors.Is(e,want){t.Fatalf("read %v, want %v",e,want)}}
  if action=="deadline"{
   p.SetReadDeadline(time.Now().Add(time.Second))
   c,e:=net.Dial("udp",p.LocalAddr().String());if e!=nil{t.Fatal(e)};defer c.Close()
   if _,e:=c.Write([]byte("ok"));e!=nil{t.Fatal(e)}
   b:=make([]byte,16);n,_,e:=p.ReadFrom(b);if e!=nil||string(b[:n])!="ok"{t.Fatalf("read after deadline %q %v",b[:n],e)}
  }
 })}
}

`)
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/udpsock_test.go"), []string{"TestUDPZeroBytePayload", "TestUDPZeroByteBuffer", "TestUDPReadSizeError", "TestUDPReadTimeout"})
	source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/udperrors/testenv"`)
	source = strings.Replace(source, "package net\n", "package udperrors\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "helpers_test.go"), `package udperrors
 import("net";"testing")
 func testableNetwork(network string)bool{return network=="udp"||network=="udp4"}
 func newLocalPacketListener(t testing.TB, network string)net.PacketConn{
  t.Helper();p,e:=net.ListenPacket(network,"127.0.0.1:0");if e!=nil{t.Fatal(e)};return p
 }
 `)
	if err := os.Mkdir(filepath.Join(dir, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "testenv/testenv.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/internal/testenv/testenv.go"), []string{"flaky", "SkipFlaky"}))
	writeFile(t, filepath.Join(dir, "testenv/LICENSE"), string(license))
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			const filter = "^(TestUDPZeroBytePayload|TestUDPZeroByteBuffer|TestUDPReadSizeError|TestUDPReadTimeout)$"
			args := []string{"-test.v", "-test.run=" + filter}
			var r programResult
			if target == "vm" {
				r = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				r = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "udp-datagram-errors", "net", filter, target, filepath.Join(runtime.GOROOT(), "src/net"), r.Text, r.Exit)
			args = []string{"-test.v", "-test.run=^(TestUDPErrorCodes|TestUDPReadError|TestUDPQueuedDatagrams|TestUDPConcurrentReads|TestUDPBlockedReads)$"}
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
