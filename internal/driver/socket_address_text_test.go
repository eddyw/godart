package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSocketAddressText(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/addresstext\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "encoding/json"
 "net"
 "os"
)

type observation struct {
 Name string
 Network string
 Text string
 Panicked bool
}

func observe(name string, addr net.Addr) (r observation) {
 r.Name = name
 r.Network = addr.Network()
 defer func() { if recover() != nil { r.Panicked = true } }()
 r.Text = addr.String()
 return
}

func main() {
 items := []struct { name string; addr net.Addr } {
  {"tcp/nil", (*net.TCPAddr)(nil)},
  {"udp/nil", (*net.UDPAddr)(nil)},
  {"tcp/zero", &net.TCPAddr{}},
  {"udp/zero", &net.UDPAddr{}},
  {"tcp/ipv4", &net.TCPAddr{IP:net.IP{127,0,0,1},Port:80}},
  {"udp/ipv4", &net.UDPAddr{IP:net.IP{127,0,0,1},Port:53}},
  {"tcp/mapped", &net.TCPAddr{IP:net.ParseIP("::ffff:127.0.0.1"),Port:80}},
  {"udp/mapped", &net.UDPAddr{IP:net.ParseIP("::ffff:127.0.0.1"),Port:53}},
  {"tcp/ipv6", &net.TCPAddr{IP:net.ParseIP("::1"),Port:80}},
  {"udp/ipv6", &net.UDPAddr{IP:net.ParseIP("::1"),Port:53}},
  {"tcp/zone", &net.TCPAddr{IP:net.ParseIP("fe80::1"),Port:80,Zone:"en0"}},
  {"udp/zone", &net.UDPAddr{IP:net.ParseIP("fe80::1"),Port:53,Zone:"en0"}},
 }
 for _, item := range items { if err:=json.NewEncoder(os.Stdout).Encode(observe(item.name,item.addr));err!=nil {panic(err)} }
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
