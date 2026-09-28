package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSystemTrustEnvironment(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/programs/systemroots")
	if err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "trust-app")
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			for _, mode := range []string{"file", "directories", "missing", "platform", "early-fallback"} {
				t.Run(mode, func(t *testing.T) {
					env := append(os.Environ(), "GART_TRUST_CASE="+mode, "GODEBUG=")
					want := runProgramEnv(t, dir, env, native)
					var got string
					if target == "vm" {
						got = runProgramEnv(t, out, env, "dart", "--packages=.dart_tool/package_config.json", "bin/main.dart")
					} else {
						got = runProgramEnv(t, out, env, filepath.Join(out, "trust-app"))
					}
					if got != want {
						t.Fatalf("system roots differ: Go %q, Gart %q", want, got)
					}
				})
			}
		})
	}
}

func TestSystemTrustInvalidCertificates(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the original assertion requires macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/trust\n\ngo 1.27\n\nrequire example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n")
	if err := os.Mkdir(filepath.Join(dir, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "testenv/testenv.go"), selectedUpstreamSource(t, "../overlay/std/internal/testenv/testenv.go", []string{"Builder", "MustHaveExec"}))
	writeFile(t, filepath.Join(dir, "testenv/exec.go"), selectedUpstreamSource(t, "../overlay/std/internal/testenv/exec.go", nil))
	writeFile(t, filepath.Join(dir, "testenv/path.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/internal/testenv/exec.go"), []string{"execPaths", "MustHaveExecPath"}))
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/crypto/x509/verify_test.go"), []string{"TestIssue51759", "macosMajorVersion", "certificateFromPEM", "googleLeaf"})
	source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
	source = strings.Replace(source, "package x509", "package x509_test\nimport . \"crypto/x509\"", 1)
	writeFile(t, filepath.Join(dir, "verify_test.go"), source)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false, Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	entry := filepath.Join(out, stats.Binaries[0].Entry)
	binary := filepath.Join(out, "trust-test")
	run(t, out, "dart", "compile", "exe", entry, "-o", binary)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			command, args := binary, upstreamTestArgs("-test.run=^TestIssue51759$")
			if target == "vm" {
				command, args = "dart", append([]string{entry}, args...)
			}
			data, err, finished := output(dir, nil, runLimit, command, args...)
			if !finished {
				t.Fatalf("certificate assertions timed out: %s", data)
			}
			code := 0
			if err != nil {
				code = 1
			}
			compareUpstream(t, "crypto/x509-system-invalid", "crypto/x509", "^TestIssue51759$", target, filepath.Join(runtime.GOROOT(), "src/crypto/x509"), string(data), code, false)
		})
	}
}
