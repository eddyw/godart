package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOSNetworkCopyOriginal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Network file copy validation requires a native filesystem")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module os_test\n\ngo 1.27\n\nrequire golang.org/x/net/nettest v0.0.0\nreplace golang.org/x/net/nettest => ./nettest\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package os_test\n")
	writeFile(t, filepath.Join(fixture, "copy_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "copy_test.go"), []string{"TestLargeCopyViaNetwork", "compareReaders", "randReader", "newRandReader", "createSocketPair"}))
	nettest := filepath.Join(runtime.GOROOT(), "src/vendor/golang.org/x/net/nettest")
	if err := os.Mkdir(filepath.Join(fixture, "nettest"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(fixture, "nettest/go.mod"), "module golang.org/x/net/nettest\n\ngo 1.27\n")
	for _, file := range []string{"nettest.go", "nettest_unix.go"} {
		writeFile(t, filepath.Join(fixture, "nettest", file), selectedUpstreamSource(t, filepath.Join(nettest, file), nil))
	}
	license, err := os.ReadFile(filepath.Join(nettest, "..", "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(fixture, "nettest/LICENSE"), string(license))
	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^TestLargeCopyViaNetwork$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "os-network-copy-original", "os", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
