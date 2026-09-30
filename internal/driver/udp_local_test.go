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
import (. "net"; "context"; "errors"; "net/netip"; "runtime"; "testing"; "time")
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
func TestDialerUDP(t *testing.T) {
 for _, family := range []struct{network, host string}{{"udp4", "127.0.0.1"}, {"udp6", "::1"}} {
  for _, network := range []string{"udp", family.network} {
   for _, api := range []string{"dial", "context", "typed"} {
    for _, localMode := range []string{"automatic", "explicit", "empty", "unspecified", "port"} {
     t.Run(network+"/"+family.host+"/"+api+"/"+localMode,func(t *testing.T) {
      server,err:=ListenUDP(family.network,&UDPAddr{IP:ParseIP(family.host)})
      if err!=nil {t.Fatal(err)}
      defer server.Close()
      var local *UDPAddr
      switch localMode {
      case "explicit":local=&UDPAddr{IP:ParseIP(family.host)}
      case "empty":local=&UDPAddr{}
      case "unspecified":local=&UDPAddr{IP:IPv4zero}
      case "port":
       reserve,err:=ListenUDP(family.network,&UDPAddr{IP:ParseIP(family.host)})
       if err!=nil {t.Fatal(err)}
       local=reserve.LocalAddr().(*UDPAddr)
       if err:=reserve.Close();err!=nil {t.Fatal(err)}
      }
      remote:=server.LocalAddr().(*UDPAddr)
      ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
      defer cancel()
      d:=Dialer{}
      if local!=nil {d.LocalAddr=local}
      var client Conn
      switch api {
      case "dial":client,err=d.Dial(network,remote.String())
      case "context":client,err=d.DialContext(ctx,network,remote.String())
      case "typed":d.LocalAddr=&TCPAddr{};client,err=d.DialUDP(ctx,network,local.AddrPort(),remote.AddrPort())
      }
      if err!=nil {t.Fatal(err)}
      defer client.Close()
      actual:=client.LocalAddr().(*UDPAddr)
      if !actual.IP.Equal(ParseIP(family.host))||actual.Port==0 {t.Fatalf("local address %v",actual)}
      if localMode=="port"&&actual.Port!=local.Port {t.Fatalf("local port %d, want %d",actual.Port,local.Port)}
      if client.RemoteAddr().String()!=remote.String(){t.Fatalf("remote %v, want %v",client.RemoteAddr(),remote)}
      if err:=client.SetWriteDeadline(time.Now().Add(5*time.Second));err!=nil {t.Fatal(err)}
      if _,err:=client.Write([]byte("hello"));err!=nil {t.Fatal(err)}
      if err:=server.SetReadDeadline(time.Now().Add(5*time.Second));err!=nil {t.Fatal(err)}
      buf:=make([]byte,10);n,from,err:=server.ReadFromUDP(buf)
      if err!=nil {t.Fatal(err)}
      if string(buf[:n])!="hello"||!from.IP.Equal(actual.IP)||from.Port!=actual.Port {t.Fatalf("source %v, local %v, payload %q",from,actual,buf[:n])}
     })
    }
   }
  }
 }
}
func TestDialerUDPCanceled(t *testing.T) {
 for _, api:=range []string{"context","typed"} {
  for _, mode:=range []string{"context","deadline","negative-timeout"} {
   t.Run(api+"/"+mode,func(t *testing.T){
    ctx:=context.Background()
    d:=Dialer{}
    want:=context.Canceled
    switch mode {
    case "context":var cancel context.CancelFunc;ctx,cancel=context.WithCancel(ctx);cancel()
    case "deadline":d.Deadline=time.Now().Add(-time.Second);want=context.DeadlineExceeded
    case "negative-timeout":d.Timeout=-time.Second;want=context.DeadlineExceeded
    }
    var c Conn
    var err error
    if api=="typed" {c,err=d.DialUDP(ctx,"udp4",netip.AddrPort{},netip.MustParseAddrPort("127.0.0.1:9"))} else {c,err=d.DialContext(ctx,"udp4","127.0.0.1:9")}
    if err==nil {c.Close();t.Fatal("canceled dial succeeded")}
    if !errors.Is(err,want) {t.Fatalf("error %v, want %v",err,want)}
    var op *OpError
    if !errors.As(err,&op)||op.Op!="dial"||op.Net!="udp4" {t.Fatalf("dial error = %T %v",err,err)}
   })
  }
 }
}
func TestDialerUDPInvalidLocal(t *testing.T) {
 for _, local:=range []Addr{&TCPAddr{},&UDPAddr{IP:IPv6loopback}} {
  d:=Dialer{LocalAddr:local}
  c,err:=d.DialContext(context.Background(),"udp4","127.0.0.1:9")
  if err==nil {c.Close();t.Fatalf("accepted local %T %v",local,local)}
  var op *OpError
  var address *AddrError
  if !errors.As(err,&op)||!errors.As(err,&address)||op.Op!="dial"||op.Net!="udp4" {t.Fatalf("dial error = %T %v",err,err)}
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
func TestUDPWildcardPeer(t *testing.T) {
 for _, tt := range []struct{network string; ip IP; host string; explicit bool}{
  {"udp", nil, "127.0.0.1", false}, {"udp", IPv4zero, "127.0.0.1", false}, {"udp", IPv6zero, "::1", false},
  {"udp4", nil, "127.0.0.1", false}, {"udp4", IPv4zero, "127.0.0.1", false},
  {"udp6", nil, "::1", false}, {"udp6", IPv4zero, "::1", false}, {"udp6", IPv6zero, "::1", false},
  {"udp", nil, "::1", true}, {"udp", IPv4zero, "::1", true},
 } {
  for _, local := range []*UDPAddr{nil, {IP: ParseIP(tt.host)}} {
   if tt.explicit && local == nil { continue }
   server, err := ListenUDP("udp", &UDPAddr{IP: ParseIP(tt.host)})
   if err != nil { t.Fatal(err) }
   remote := server.LocalAddr().(*UDPAddr)
   client, err := DialUDP(tt.network, local, &UDPAddr{IP: tt.ip, Port: remote.Port})
   if err != nil { server.Close(); t.Fatal(err) }
   if client.RemoteAddr().String() != remote.String() || !client.LocalAddr().(*UDPAddr).IP.Equal(remote.IP) {
    t.Fatalf("%s %v: local=%v remote=%v, want peer %v", tt.network, tt.ip, client.LocalAddr(), client.RemoteAddr(), remote)
   }
   client.SetDeadline(time.Now().Add(time.Second))
   server.SetDeadline(time.Now().Add(time.Second))
   if _, err := client.Write([]byte("ping")); err != nil { t.Fatal(err) }
   var b [4]byte
   n, from, err := server.ReadFromUDP(b[:])
   if n != 4 || err != nil || string(b[:]) != "ping" { t.Fatalf("server read: %d %v %q", n, err, b) }
   if _, err := server.WriteToUDP([]byte("pong"), from); err != nil { t.Fatal(err) }
   n, err = client.Read(b[:])
   if n != 4 || err != nil || string(b[:]) != "pong" { t.Fatalf("client read: %d %v %q", n, err, b) }
   client.Close()
   server.Close()
  }
 }
}

func TestUDPIPv4LocalZone(t *testing.T) {
 for _, network := range []string{"udp", "udp4"} {
  server, err := ListenUDP(network, &UDPAddr{IP: IPv4(127,0,0,1)})
  if err != nil { t.Fatal(err) }
  defer server.Close()
  remote := server.LocalAddr().(*UDPAddr)
  for _, ip := range []IP{nil, IPv4zero, IPv4(127,0,0,1)} {
   for _, api := range []string{"package", "dial", "context"} {
    local := &UDPAddr{IP: ip, Zone: "invalid-interface"}
    var c Conn
    if api == "package" { c, err = DialUDP(network, local, remote) } else {
     d := Dialer{LocalAddr: local}
     if api == "dial" { c, err = d.Dial(network, remote.String()) } else { c, err = d.DialContext(context.Background(), network, remote.String()) }
    }
    if err != nil { t.Fatalf("%s %s %v: %v", network, api, ip, err) }
    actual := c.LocalAddr().(*UDPAddr)
    if actual.Zone != "" || !actual.IP.Equal(remote.IP) { t.Fatalf("incorrect local address: %v", actual) }
    c.SetWriteDeadline(time.Now().Add(time.Second))
    server.SetReadDeadline(time.Now().Add(time.Second))
    if _, err := c.Write([]byte("ok")); err != nil { t.Fatal(err) }
    var b [2]byte
    n, from, err := server.ReadFromUDP(b[:])
    if n != 2 || err != nil || string(b[:]) != "ok" || from.String() != actual.String() { t.Fatalf("datagram: %d %v %v %q", n, from, err, b) }
    c.Close()
   }
  }
 }
}

func TestUDPIPv4RemoteZone(t *testing.T) {
 for _, network := range []string{"udp", "udp4"} {
  server, err := ListenUDP(network, &UDPAddr{IP: IPv4(127,0,0,1)})
  if err != nil { t.Fatal(err) }
  defer server.Close()
  expected := server.LocalAddr().(*UDPAddr)
  for _, ip := range []IP{nil, IPv4zero, IPv4(127,0,0,1)} {
   for _, zone := range []string{"1", "invalid-interface"} {
    remote := &UDPAddr{IP: ip, Port: expected.Port, Zone: zone}
    c, err := DialUDP(network, nil, remote)
    if err != nil { t.Fatalf("%s %v: %v", network, remote, err) }
    actual := c.RemoteAddr().(*UDPAddr)
    if actual.Zone != "" || actual.String() != expected.String() { t.Fatalf("peer = %v, want %v", actual, expected) }
    c.SetDeadline(time.Now().Add(time.Second))
    server.SetDeadline(time.Now().Add(time.Second))
    if _, err := c.Write([]byte("ok")); err != nil { t.Fatal(err) }
    var b [2]byte
    n, from, err := server.ReadFromUDP(b[:])
    if n != 2 || err != nil || string(b[:]) != "ok" { t.Fatalf("read: %d %v %q", n, err, b) }
    if _, err := server.WriteToUDP([]byte("hi"), from); err != nil { t.Fatal(err) }
    n, err = c.Read(b[:])
    if n != 2 || err != nil || string(b[:]) != "hi" { t.Fatalf("reply: %d %v %q", n, err, b) }
    c.Close()
   }
  }
 }
}

func TestUDPZeroPortPeer(t *testing.T) {
 if runtime.GOOS != "linux" { t.Skip("Linux permits a UDP peer with port 0") }
 for _, host := range []string{"127.0.0.1", "::1"} {
  remote := &UDPAddr{IP: ParseIP(host)}
  c, err := DialUDP("udp", nil, remote)
  if err != nil { t.Fatal(err) }
  if c.RemoteAddr().String() != remote.String() { t.Fatalf("remote = %v, want %v", c.RemoteAddr(), remote) }
  if err := c.Close(); err != nil { t.Fatal(err) }
 }
}

`)
	t.Log(run(t, dir, "go", "test", "-short="+strconv.FormatBool(testing.Short()), "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			args := upstreamTestArgs("-test.run=^TestUDPConnLocalAndRemoteNames$")
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "udp-local", "net", "^TestUDPConnLocalAndRemoteNames$", target, filepath.Join(runtime.GOROOT(), "src/net"), result.Text, result.Exit, false)
			if target == "vm" {
				t.Log(run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.short="+strconv.FormatBool(testing.Short()), "-test.run=^Test(LocalBind|CloseRebind|DialerUDP|DialerUDPCanceled|DialerUDPInvalidLocal|UDPWildcardPeer|UDPZeroPortPeer|UDPIPv4LocalZone|UDPIPv4RemoteZone)$"))
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=^Test(LocalBind|CloseRebind|DialerUDP|DialerUDPCanceled|DialerUDPInvalidLocal|UDPWildcardPeer|UDPZeroPortPeer|UDPIPv4LocalZone|UDPIPv4RemoteZone)$"})
				t.Log(result.Text)
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
