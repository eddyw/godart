package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOpenFileFlags(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native file flag validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/openflags\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "flags_test.go"), `package openflags

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFlags(t *testing.T) {
	t.Run("read-create", func(t *testing.T) {
		f, e := os.OpenFile(filepath.Join(t.TempDir(), "f"), os.O_RDONLY|os.O_CREATE, 0600)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if n, e := f.Read(make([]byte, 1)); n != 0 || e != io.EOF {
			t.Fatalf("read = %d %v", n, e)
		}
	})
	t.Run("write-only", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "f")
		if e := os.WriteFile(name, []byte("old"), 0600); e != nil {
			t.Fatal(e)
		}
		f, e := os.OpenFile(name, os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if n, e := f.Read(make([]byte, 3)); n != 0 || e == nil {
			t.Fatalf("write-only read = %d %v", n, e)
		}
	})
	t.Run("append-read", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "f")
		if e := os.WriteFile(name, []byte("old"), 0600); e != nil {
			t.Fatal(e)
		}
		f, e := os.OpenFile(name, os.O_RDWR|os.O_APPEND, 0600)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		b := make([]byte, 3)
		if n, e := f.Read(b); n != 3 || e != nil || string(b) != "old" {
			t.Fatalf("append read = %d %v %q", n, e, b)
		}
		if _, e := f.Seek(0, 0); e != nil {
			t.Fatal(e)
		}
		if _, e := f.Write([]byte("new")); e != nil {
			t.Fatal(e)
		}
		b, e = os.ReadFile(name)
		if e != nil || string(b) != "oldnew" {
			t.Fatalf("append after seek = %q %v", b, e)
		}
	})
	t.Run("append-writeat", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "f")
		f, e := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		for _, b := range [][]byte{nil, []byte("x")} {
			n, e := f.WriteAt(b, 0)
			if n != 0 || e == nil || e.Error() != "os: invalid use of WriteAt on file opened with O_APPEND" {
				t.Fatalf("append WriteAt = %d %v", n, e)
			}
		}
	})
	t.Run("empty-writeat", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "f")
		f, e := os.OpenFile(name, os.O_CREATE|os.O_RDONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if n, e := f.WriteAt(nil, 1); n != 0 || e != nil {
			t.Fatalf("empty WriteAt = %d %v", n, e)
		}
		if n, e := f.WriteAt(nil, -1); n != 0 || e == nil {
			t.Fatalf("negative empty WriteAt = %d %v", n, e)
		}
	})
	t.Run("sync", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "f")
		f, e := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0600)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if n, e := f.Write([]byte("sync")); n != 4 || e != nil {
			t.Fatalf("synchronous write = %d %v", n, e)
		}
	})

}
`)
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := []string{"-test.v"}
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			t.Log(result.Text)
			if result.Exit != 0 {
				t.Fatal(result.Text)
			}
		})
	}
}
