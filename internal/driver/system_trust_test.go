package driver_test

import (
	"os"
	"path/filepath"
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
