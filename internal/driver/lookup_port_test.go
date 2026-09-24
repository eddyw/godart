package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLookupPortUpstream(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	netDir := filepath.Join(runtime.GOROOT(), "src", "net")
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/lookupport\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(netDir, "lookup_test.go"), []string{"TestLookupPort_Minimal"})
	source = strings.Replace(source, "package net\n", "package net_test\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "lookup_test.go"), source)
	writeFile(t, filepath.Join(dir, "errors_test.go"), `package net_test
import ("context"; "errors"; "net"; "testing")
func TestGoResolverPortErrors(t *testing.T) {
 r := net.Resolver{PreferGo: true}
 for _, network := range []string{"tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "ip", ""} {
  for _, service := range []string{"--badport--", "bad\x00port"} {
   t.Run(network+"/"+service, func(t *testing.T) {
    port, err := r.LookupPort(context.Background(), network, service)
    name := network
    switch network { case "tcp4", "tcp6": name = "tcp"; case "udp4", "udp6": name = "udp"; case "": name = "ip" }
    var dns *net.DNSError
    if port != 0 || !errors.As(err, &dns) { t.Fatalf("LookupPort = %d, %T %v", port, err, err) }
    if dns.Name != name+"/"+service || dns.Err != "unknown port" || !dns.IsNotFound || dns.Timeout() || dns.Temporary() { t.Fatalf("error = %+v", dns) }
   })
  }
 }
}
`)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	t.Log(run(t, dir, "go", "test", "-run", "^TestGoResolverPortErrors$", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			runTarget := func(dir string, args []string) programResult {
				if target == "vm" {
					return programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
				}
				return runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			result := runTarget(dir, []string{"-test.v", "-test.run=^TestGoResolverPortErrors$"})
			t.Log(result.Text)
			if result.Exit != 0 {
				t.Fatal(result.Text)
			}
			const filter = "^TestLookupPort_Minimal$"
			result = runTarget(netDir, []string{"-test.v", "-test.run=" + filter})
			compareUpstream(t, "net-lookup-port", "net", filter, target, netDir, result.Text, result.Exit)
		})
	}
}
