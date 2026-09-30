package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
import("errors";"fmt";"net";"net/netip";"os";"runtime";"syscall";"testing";"time")
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
 for attempt:=0;attempt<100;attempt++ {t.Run(fmt.Sprint(attempt),func(t *testing.T){
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
 })}
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

func TestConnectedWriteDestination(t *testing.T) {
 listener, err := net.ListenPacket("udp", "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 defer listener.Close()
 remote := listener.LocalAddr().(*net.UDPAddr)
 connection, err := net.DialUDP("udp", nil, remote)
 if err != nil { t.Fatal(err) }
 defer connection.Close()
 for _, test := range []struct{name string; write func([]byte)(int,error)}{
  {"WriteTo",func(b []byte)(int,error){return connection.WriteTo(b,remote)}},
  {"WriteToUDP",func(b []byte)(int,error){return connection.WriteToUDP(b,remote)}},
  {"WriteToUDPAddrPort",func(b []byte)(int,error){return connection.WriteToUDPAddrPort(b,remote.AddrPort())}},
 } {
  t.Run(test.name,func(t *testing.T){
   n, err := test.write([]byte("payload"))
   if n != 0 || !errors.Is(err,net.ErrWriteToConnected) { t.Fatalf("write = %d %v",n,err) }
   var op *net.OpError
   if !errors.As(err,&op) { t.Fatalf("error type = %T, want *net.OpError",err) }
   if op.Op != "write" || op.Net != "udp" || op.Source == nil || op.Addr == nil { t.Fatalf("operation = %#v",op) }
   if op.Source.String()!=connection.LocalAddr().String() || op.Addr.String()!=remote.String() { t.Fatalf("addresses = %#v",op) }
   if op.Err != net.ErrWriteToConnected { t.Fatalf("cause = %T %v",op.Err,op.Err) }
  })
 }
}

func TestUDPDestinationErrors(t *testing.T) {
 server,err:=net.ListenUDP("udp",&net.UDPAddr{IP:net.IPv4(127,0,0,1)})
 if err!=nil{t.Fatal(err)};defer server.Close()
 remote:=server.LocalAddr().(*net.UDPAddr)
 if len(remote.IP)!=net.IPv4len{t.Fatalf("socket IPv4 length = %d, want %d",len(remote.IP),net.IPv4len)}
 for _,state:=range []string{"connected","bound","closed-connected","closed-bound","zero","nil"}{
  t.Run(state,func(t *testing.T){
   var c *net.UDPConn
   connected:=state=="connected"||state=="closed-connected"
   switch state{
   case "connected","closed-connected":c,err=net.DialUDP("udp",nil,remote)
   case "bound","closed-bound":c,err=net.ListenUDP("udp",&net.UDPAddr{IP:net.IPv4(127,0,0,1)})
   case "zero":c=new(net.UDPConn)
   }
   if err!=nil{t.Fatal(err)}
   empty:=state=="zero"||state=="nil"
   local:=""
   if !empty{local=c.LocalAddr().String();defer c.Close()}
   if state=="closed-connected"||state=="closed-bound"{c.Close()}
   missing:=errors.New("missing address")
   for _,method:=range []string{"WriteTo/nil","WriteTo/typed-nil","WriteTo/wrong-type","WriteToUDP/nil","WriteToUDPAddrPort/invalid","WriteToUDPAddrPort/valid"}{
    t.Run(method,func(t *testing.T){
     defer func(){
      p:=recover()
      if state=="nil"{if p==nil{t.Error("nil receiver did not panic")};return}
      if p!=nil{t.Errorf("panic: %v",p)}
     }()
     var n int;var err error
     switch method{
     case "WriteTo/nil":n,err=c.WriteTo(nil,nil)
     case "WriteTo/typed-nil":n,err=c.WriteTo(nil,(*net.UDPAddr)(nil))
     case "WriteTo/wrong-type":n,err=c.WriteTo(nil,&net.TCPAddr{IP:remote.IP,Port:remote.Port})
     case "WriteToUDP/nil":n,err=c.WriteToUDP(nil,nil)
     case "WriteToUDPAddrPort/invalid":n,err=c.WriteToUDPAddrPort(nil,netip.AddrPort{})
     case "WriteToUDPAddrPort/valid":n,err=c.WriteToUDPAddrPort(nil,remote.AddrPort())
     }
     if n!=0{t.Fatalf("count = %d, want 0",n)}
     if empty{if err!=syscall.EINVAL{t.Fatalf("error = %T %v, want EINVAL",err,err)};return}
     var want error
     switch{
     case method=="WriteTo/nil"||method=="WriteTo/wrong-type":want=syscall.EINVAL
     case connected:want=net.ErrWriteToConnected
     case method!="WriteToUDPAddrPort/valid":want=missing
     case state=="closed-bound":want=net.ErrClosed
     }
     if want==nil{if err!=nil{t.Fatal(err)};return}
     op,ok:=err.(*net.OpError)
     if !ok{t.Fatalf("error = %T %v, want OpError",err,err)}
     if op.Op!="write"||op.Net!="udp"||op.Source==nil||op.Source.String()!=local{t.Fatalf("operation = %#v",op)}
     if want==missing{
      if op.Err.Error()!=want.Error(){t.Fatalf("cause = %v, want %v",op.Err,want)}
     }else if op.Err!=want||!errors.Is(err,want){t.Fatalf("cause = %v, want %v",op.Err,want)}
     addrType,addr:="<nil>",""
     switch method{
     case "WriteTo/wrong-type":addrType,addr="*net.TCPAddr",remote.String()
     case "WriteToUDPAddrPort/invalid":addrType,addr="net.addrPortUDPAddr","invalid AddrPort"
     case "WriteToUDPAddrPort/valid":addrType,addr="net.addrPortUDPAddr",remote.String()
     }
     if got:=fmt.Sprintf("%T",op.Addr);got!=addrType{t.Fatalf("destination type = %s, want %s",got,addrType)}
     if op.Addr!=nil&&op.Addr.String()!=addr{t.Fatalf("destination = %v, want %s",op.Addr,addr)}
     text:="write udp "+local
     if addr!=""{text+="->"+addr}
     text+=": "+want.Error()
     if err.Error()!=text{t.Fatalf("error = %q, want %q",err.Error(),text)}
    })
   }
  })
 }
}

func TestUDPWritePorts(t *testing.T) {
 for _, host := range []string{"127.0.0.1", "::1"} {
  server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host)})
  if err != nil { t.Fatal(err) }
  defer server.Close()
  for _, state := range []string{"bound", "connected", "closed", "deadline"} {
   var c *net.UDPConn
   if state == "connected" {
    c, err = net.DialUDP("udp", nil, server.LocalAddr().(*net.UDPAddr))
   } else {
    c, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host)})
   }
   if err != nil { t.Fatal(err) }
   local := c.LocalAddr().String()
   if state == "closed" { c.Close() }
   if state == "deadline" { c.SetWriteDeadline(time.Unix(1, 0)) }
   for _, port := range []int{-1, 65536} {
    remote := &net.UDPAddr{IP: net.ParseIP(host), Port: port}
    for _, api := range []string{"WriteTo", "WriteToUDP"} {
     var n int
     if api == "WriteTo" { n, err = c.WriteTo(nil, remote) } else { n, err = c.WriteToUDP(nil, remote) }
     if n != 0 { t.Fatalf("%s %s %d: wrote %d bytes", state, api, port, n) }
     var op *net.OpError
     if !errors.As(err, &op) || op.Op != "write" || op.Net != "udp" || op.Source.String() != local || op.Addr != remote {
      t.Fatalf("%s %s %d: incorrect error metadata: %v", state, api, port, err)
     }
     switch state {
     case "connected":
      if !errors.Is(err, net.ErrWriteToConnected) { t.Fatalf("incorrect connected error: %v", err) }
     case "closed":
      if !errors.Is(err, net.ErrClosed) { t.Fatalf("incorrect closed error: %v", err) }
     case "deadline":
      if !errors.Is(err, os.ErrDeadlineExceeded) { t.Fatalf("incorrect deadline error: %v", err) }
     default:
      var call *os.SyscallError
      if !errors.Is(err, syscall.EINVAL) || !errors.As(err, &call) || call.Syscall != "sendto" {
       t.Fatalf("%s %d: incorrect port error: %v", api, port, err)
      }
     }
    }
   }
   if state == "bound" {
    if n, err := c.WriteToUDP([]byte("ok"), server.LocalAddr().(*net.UDPAddr)); n != 2 || err != nil { t.Fatalf("valid write: %d %v", n, err) }
    server.SetReadDeadline(time.Now().Add(time.Second))
    var b [2]byte
    if n, _, err := server.ReadFromUDP(b[:]); n != 2 || err != nil || string(b[:]) != "ok" { t.Fatalf("valid read: %d %v %q", n, err, b) }
   }
   c.Close()
  }
 }
}

func TestUDPWriteAddresses(t *testing.T) {
 for _, network := range []string{"udp4", "udp6"} {
  local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}
  want := "non-IPv4 address"
  if network == "udp6" { local.IP, want = net.IPv6loopback, "non-IPv6 address" }
  server, err := net.ListenUDP(network, local)
  if err != nil { t.Fatal(err) }
  defer server.Close()
  ips := []net.IP{{1}, {1, 2, 3}, {1, 2, 3, 4, 5}, make(net.IP, 17)}
  if network == "udp4" { ips = append(ips, net.IPv6loopback) }
  for _, state := range []string{"bound", "connected", "closed", "deadline"} {
   var c *net.UDPConn
   if state == "connected" { c, err = net.DialUDP(network, nil, server.LocalAddr().(*net.UDPAddr)) } else { c, err = net.ListenUDP(network, local) }
   if err != nil { t.Fatal(err) }
   source := c.LocalAddr().String()
   if state == "closed" { c.Close() }
   if state == "deadline" { c.SetWriteDeadline(time.Unix(1, 0)) }
   for _, ip := range ips {
    for _, port := range []int{9, -1} {
     addr := &net.UDPAddr{IP: ip, Port: port}
     for _, api := range []string{"WriteTo", "WriteToUDP"} {
      var n int
      if api == "WriteTo" { n, err = c.WriteTo(nil, addr) } else { n, err = c.WriteToUDP(nil, addr) }
      var op *net.OpError
      if n != 0 || !errors.As(err, &op) || op.Op != "write" || op.Net != network || op.Addr != addr || op.Source.String() != source {
       t.Fatalf("%s %s %s: incorrect result: %d %v", network, state, api, n, err)
      }
      if state == "connected" {
       if !errors.Is(err, net.ErrWriteToConnected) { t.Fatalf("incorrect connected error: %v", err) }
      } else {
       var cause *net.AddrError
       if !errors.As(err, &cause) || cause.Err != want || cause.Addr != ip.String() {
        t.Fatalf("%s %s %s: incorrect address error: %v", network, state, api, err)
       }
      }
     }
    }
   }
   c.Close()
  }
 }
}

func TestUDPWildcardDestination(t *testing.T) {
 for _, host := range []string{"127.0.0.1", "::1"} {
  server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host)})
  if err != nil { t.Fatal(err) }
  defer server.Close()
  client, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host)})
  if err != nil { t.Fatal(err) }
  defer client.Close()
  ips := []net.IP{nil, net.IPv4zero}
  if host == "::1" { ips = append(ips, net.IPv6zero) }
  for _, ip := range ips {
   addr := &net.UDPAddr{IP: ip, Port: server.LocalAddr().(*net.UDPAddr).Port}
   for _, api := range []string{"WriteTo", "WriteToUDP"} {
    client.SetWriteDeadline(time.Now().Add(time.Second))
    server.SetReadDeadline(time.Now().Add(time.Second))
    var n int
    if api == "WriteTo" { n, err = client.WriteTo([]byte("ok"), addr) } else { n, err = client.WriteToUDP([]byte("ok"), addr) }
    if client.RemoteAddr() != nil { t.Fatal("unconnected socket acquired a peer") }
    if runtime.GOOS == "darwin" {
     var op *net.OpError
     var call *os.SyscallError
     if n != 0 || !errors.Is(err, syscall.EHOSTUNREACH) || !errors.As(err, &op) || op.Op != "write" || op.Net != "udp" || op.Addr != addr || op.Source.String() != client.LocalAddr().String() || !errors.As(err, &call) || call.Syscall != "sendto" {
      t.Fatalf("%s %s %v: incorrect wildcard error: %d %v", host, api, ip, n, err)
     }
     continue
    }
    if n != 2 || err != nil { t.Fatalf("%s %s %v: write = %d %v", host, api, ip, n, err) }
    var b [2]byte
    n, from, err := server.ReadFromUDP(b[:])
    if n != 2 || err != nil || string(b[:]) != "ok" || from.String() != client.LocalAddr().String() {
     t.Fatalf("%s %s %v: read = %d %v %v %q", host, api, ip, n, from, err, b)
    }
   }
  }
 }
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
	t.Log(run(t, dir, "go", "test", "-short="+strconv.FormatBool(testing.Short()), "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			const filter = "^(TestUDPZeroBytePayload|TestUDPZeroByteBuffer|TestUDPReadSizeError|TestUDPReadTimeout)$"
			args := upstreamTestArgs("-test.run=" + filter)
			var r programResult
			if target == "vm" {
				r = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				r = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "udp-datagram-errors", "net", filter, target, filepath.Join(runtime.GOROOT(), "src/net"), r.Text, r.Exit, false)
			args = []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=^(TestUDPErrorCodes|TestUDPReadError|TestUDPQueuedDatagrams|TestUDPConcurrentReads|TestUDPBlockedReads|TestConnectedWriteDestination|TestUDPDestinationErrors|TestUDPWritePorts|TestUDPWriteAddresses|TestUDPWildcardDestination)$"}
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
