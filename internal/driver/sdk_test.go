package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDartSDK(t *testing.T) {
	for _, name := range []string{"standalone", "flutter", "fvm", "missing"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			launcher := filepath.Join(bin, "dart")
			putKernelFile(t, launcher, "#!/bin/sh\nexit 0\n")
			if err := os.Chmod(launcher, 0755); err != nil {
				t.Fatal(err)
			}
			sdk := root
			if name == "flutter" || name == "fvm" {
				sdk = filepath.Join(bin, "cache", "dart-sdk")
			}
			if name != "missing" {
				putKernelFile(t, filepath.Join(sdk, "version"), "3.13.4")
			}
			if name == "fvm" {
				alias := filepath.Join(t.TempDir(), "default")
				if err := os.Symlink(root, alias); err != nil {
					t.Fatal(err)
				}
				bin = filepath.Join(alias, "bin")
			}
			t.Setenv("PATH", bin)
			got, err := dartSDK()
			if name == "missing" {
				if err == nil || !strings.Contains(err.Error(), "cannot locate the Dart SDK") {
					t.Fatalf("SDK %q, error %v", got, err)
				}
				return
			}
			want, resolveErr := filepath.EvalSymlinks(sdk)
			if resolveErr != nil {
				t.Fatal(resolveErr)
			}
			if err != nil || got != want {
				t.Fatalf("SDK %q, want %q, error %v", got, want, err)
			}
		})
	}
}
