package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestSocketDeadlineEffects(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/deadlineeffects\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "fixture_test.go"), `package deadlineeffects
import("errors";"io";"net";"os";"testing";"time")
func tcpPair(t *testing.T)(net.Conn,net.Conn){
 t.Helper()
 l,e:=net.Listen("tcp","127.0.0.1:0");if e!=nil{t.Fatal(e)}
 defer l.Close()
 c,e:=net.Dial("tcp",l.Addr().String());if e!=nil{t.Fatal(e)}
 t.Cleanup(func(){c.Close()})
 p,e:=l.Accept();if e!=nil{t.Fatal(e)}
 t.Cleanup(func(){p.Close()})
 return c,p
}
func udpPair(t *testing.T)(*net.UDPConn,*net.UDPConn){
 t.Helper()
 p,e:=net.ListenUDP("udp",&net.UDPAddr{IP:net.IPv4(127,0,0,1)});if e!=nil{t.Fatal(e)}
 t.Cleanup(func(){p.Close()})
 c,e:=net.DialUDP("udp",nil,p.LocalAddr().(*net.UDPAddr));if e!=nil{t.Fatal(e)}
 t.Cleanup(func(){c.Close()})
 return c,p
}
func wantTimeout(t *testing.T,n int,e error){
 t.Helper()
 if n!=0||!errors.Is(e,os.ErrDeadlineExceeded){t.Errorf("result %d %v, want 0 and deadline exceeded",n,e)}
}
func TestDeadlineEffects(t *testing.T){
 t.Run("tcp-write",func(t *testing.T){
  c,p:=tcpPair(t)
  if e:=c.SetWriteDeadline(time.Unix(1,0));e!=nil{t.Fatal(e)}
  n,e:=c.Write(nil);wantTimeout(t,n,e)
  n,e=c.Write([]byte("expired"));wantTimeout(t,n,e)
  p.SetReadDeadline(time.Now().Add(20*time.Millisecond))
  b:=make([]byte,32);n,e=p.Read(b);wantTimeout(t,n,e)
 })
 t.Run("tcp-buffered-read",func(t *testing.T){
  c,p:=tcpPair(t)
  if _,e:=c.Write([]byte("ab"));e!=nil{t.Fatal(e)}
  p.SetReadDeadline(time.Now().Add(time.Second))
  b:=make([]byte,1);if _,e:=io.ReadFull(p,b);e!=nil{t.Fatal(e)}
  if string(b)!="a"{t.Fatalf("first byte %q",b)}
  p.SetReadDeadline(time.Unix(1,0))
  if n,e:=p.Read(nil);n!=0||e!=nil{t.Errorf("empty read %d %v",n,e)}
  n,e:=p.Read(b);wantTimeout(t,n,e)
  p.SetReadDeadline(time.Now().Add(time.Second))
  if _,e:=io.ReadFull(p,b);e!=nil{t.Fatal(e)}
  if string(b)!="b"{t.Fatalf("remaining byte %q",b)}
 })
 t.Run("udp-write",func(t *testing.T){
  c,p:=udpPair(t)
  c.SetWriteDeadline(time.Unix(1,0))
  n,e:=c.Write([]byte("expired"));wantTimeout(t,n,e)
  p.SetReadDeadline(time.Now().Add(20*time.Millisecond))
  b:=make([]byte,32);n,_,e=p.ReadFrom(b);wantTimeout(t,n,e)
  p.SetWriteDeadline(time.Unix(1,0))
  n,e=p.WriteTo([]byte("expired"),c.LocalAddr());wantTimeout(t,n,e)
  c.SetReadDeadline(time.Now().Add(20*time.Millisecond))
  n,e=c.Read(b);wantTimeout(t,n,e)
 })
 t.Run("udp-buffered-read",func(t *testing.T){
  c,p:=udpPair(t)
  if _,e:=c.Write([]byte("datagram"));e!=nil{t.Fatal(e)}
  p.SetReadDeadline(time.Unix(1,0))
  if n,e:=p.Read(nil);n!=0||e!=nil{t.Errorf("empty UDP read %d %v",n,e)}
  b:=make([]byte,32);n,_,e:=p.ReadFrom(b);wantTimeout(t,n,e)
  p.SetReadDeadline(time.Now().Add(time.Second))
  n,_,e=p.ReadFrom(b);if e!=nil{t.Fatal(e)}
  if string(b[:n])!="datagram"{t.Fatalf("datagram %q",b[:n])}
  if e:=p.Close();e!=nil{t.Fatal(e)}
  if n,e:=p.Read(nil);n!=0||!errors.Is(e,net.ErrClosed){t.Errorf("closed empty UDP read %d %v",n,e)}
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
