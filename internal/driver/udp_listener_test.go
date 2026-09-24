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

func TestUDPListenerPortOwnership(t *testing.T) {
	probe, err := net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6loopback})
	if err != nil {
		t.Skipf("IPv6 loopback is unavailable: %v", err)
	}
	probe.Close()
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udplistener\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/listen_test.go"), []string{"udpListenerTests", "checkSecondListener"})
	source = strings.Replace(source, "package net\n", "package udplistener\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "ownership_test.go"), `package udplistener
import (. "net"; "testing")
func TestPortOwnership(t *testing.T) {
 for _, tt := range udpListenerTests {
  t.Run(tt.network+"/"+tt.address, func(t *testing.T) {
   addr,err:=ResolveUDPAddr(tt.network,JoinHostPort(tt.address,"0"))
   if err!=nil {t.Fatal(err)}
   first,err:=ListenUDP(tt.network,addr)
   if err!=nil {t.Fatal(err)}
   defer first.Close()
   taken:=first.LocalAddr().(*UDPAddr)
   second,err:=ListenUDP(tt.network,taken)
   if err==nil {second.Close()}
   if err:=checkSecondListener(tt.network,tt.address,err);err!=nil {t.Fatal(err)}
   if err:=first.Close();err!=nil {t.Fatal(err)}
   next,err:=ListenUDP(tt.network,taken)
   if err!=nil {t.Fatal(err)}
   next.Close()
  })
 }
}
`)
	run(t, dir, "go", "test", "-v", ".")
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			if target == "vm" {
				run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v")
			} else {
				result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v"})
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
