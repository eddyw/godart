package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUnsupportedFIPSMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fipsmode\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"crypto/fips140"
	"fmt"
)
func main() { fmt.Println(fips140.Enabled(), fips140.Enforced()) }
`)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, setting := range []string{"", "off", "on", "only", "debug"} {
		t.Run(setting, func(t *testing.T) {
			t.Setenv("GODEBUG", "fips140="+setting)
			result := programOutcome(t, out, "dart", "bin/main.dart")
			if setting == "" || setting == "off" {
				if result.Exit != 0 || result.Text != "false false" {
					t.Fatalf("default crypto failed: %+v", result)
				}
			} else if result.Exit == 0 || !strings.Contains(result.Text, "Dart does not support FIPS 140-3 mode") {
				t.Fatalf("unsupported FIPS mode did not fail explicitly: %+v", result)
			}
		})
	}
}
