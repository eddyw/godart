package driver_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestInterfaceUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native interface metadata requires macOS or Linux")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/net")
	names := map[string][]string{
		"interface_test.go":       {"TestInterfaces", "TestInterfaceAddrs", "TestInterfaceUnicastAddrs", "TestInterfaceMulticastAddrs", "ifStats", "interfaceStats", "routeStats", "validateInterfaceUnicastAddrs", "validateInterfaceMulticastAddrs", "checkUnicastStats", "checkMulticastStats"},
		"interface_linux_test.go": {"numOfTestIPv4MCAddrs", "numOfTestIPv6MCAddrs", "igmpInterfaceTable", "igmp6InterfaceTable", "TestParseProcNet"},
	}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		if declarations, ok := names[filepath.Base(path)]; ok {
			sources[path] = []byte(selectedUpstreamSource(t, path, declarations))
		} else {
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + file.Name.Name + "\n")
		}
	}
	sources[filepath.Join(dir, "platform_test.go")] = []byte(`package net
func supportsIPv4() bool { return supportsTCPFamily("tcp4", "127.0.0.1:0") }
func supportsIPv6() bool { return supportsTCPFamily("tcp6", "[::1]:0") }
func supportsTCPFamily(network, address string) bool {
 listener, err := Listen(network, address)
 if err != nil { return false }
 listener.Close()
 return true
}
`)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	filter := "^(TestInterfaces|TestInterfaceAddrs|TestInterfaceUnicastAddrs|TestInterfaceMulticastAddrs"
	if runtime.GOOS == "linux" {
		filter += "|TestParseProcNet"
	}
	filter += ")$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "net-interfaces", "net", filter, target, dir, result.Text, result.Exit)
		})
	}
}
