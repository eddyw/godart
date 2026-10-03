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

func TestUnixgramLinuxAddressOriginal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Original abstract Unix datagrams require Linux")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/net")
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		switch filepath.Base(path) {
		case "unixsock_linux_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestUnixgramAutobind", "TestUnixgramLinuxAbstractLongName"}))
		case "platform_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"testableNetwork"}))
		case "platform_unix_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"unixEnabledOnAIX", "init", "supportsUnixSocket"}))
		default:
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + file.Name.Name + "\n")
		}
	}
	sources[filepath.Join(dir, "unixgram_probe_test.go")] = []byte(`package net
func supportsIPv4() bool {
 c, err := ListenUDP("udp4", &UDPAddr{IP: IPv4(127,0,0,1)})
 if err != nil { return false }; c.Close(); return true
}
func supportsIPv6() bool {
 c, err := ListenUDP("udp6", &UDPAddr{IP: IPv6loopback})
 if err != nil { return false }; c.Close(); return true
}
`)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^(TestUnixgramAutobind|TestUnixgramLinuxAbstractLongName)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "unixgram-linux-address-original", "net", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
