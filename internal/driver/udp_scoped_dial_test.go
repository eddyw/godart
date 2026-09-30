package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUDPScopedDialErrors(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udpscoped\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("context";"fmt";"net";"errors")
func main(){
 ctx,cancel:=context.WithCancel(context.Background());cancel()
 for _,network:=range []string{"udp","udp6"}{
  for _,host:=range []string{"::1","2001:db8::1","fe80::1","::"}{
   for _,zone:=range []string{"","0","999999","no-such-interface"}{
    address:=host;if zone!=""{address+="%"+zone}
    c,err:=new(net.Dialer).DialContext(ctx,network,net.JoinHostPort(address,"9"))
    if c!=nil{c.Close();panic("canceled dial returned a connection")}
    var op *net.OpError
    if !errors.As(err,&op){panic(err)}
    fmt.Printf("%s %s canceled=%t source=%v addr=%v\n",network,address,errors.Is(err,context.Canceled),op.Source,op.Addr)
   }
  }
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
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
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

func TestUDPScopedDestination(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udpscopedlive\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("context";"fmt";"net";"runtime";"time")
func main(){
 ip:="127.0.0.1"
 // macOS rejects a bind to an unconfigured loopback address.
 if runtime.GOOS=="linux"{ip="127.0.0.2"}
 for _,zone:=range []string{"","0","no-such-interface"}{
  for _,api:=range []string{"dial","context"}{
   server,err:=net.ListenUDP("udp4",&net.UDPAddr{IP:net.ParseIP(ip)})
   if err!=nil{panic(err)}
   host:="::ffff:"+ip;if zone!=""{host+="%"+zone}
   address:=net.JoinHostPort(host,fmt.Sprint(server.LocalAddr().(*net.UDPAddr).Port))
   var client net.Conn
   if api=="dial"{client,err=net.Dial("udp",address)}else{client,err=new(net.Dialer).DialContext(context.Background(),"udp",address)}
   if err!=nil{panic(err)}
   if _,err=client.Write([]byte("hello"));err!=nil{panic(err)}
   if err=server.SetReadDeadline(time.Now().Add(2*time.Second));err!=nil{panic(err)}
   data:=make([]byte,32);n,_,err:=server.ReadFromUDP(data);if err!=nil{panic(err)}
   fmt.Printf("%s/%s remote=%s payload=%t\n",api,zone,client.RemoteAddr().(*net.UDPAddr).IP,n==5&&string(data[:n])=="hello")
   client.Close();server.Close()
  }
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
