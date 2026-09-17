package load

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func surfaceFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSurfaceCacheReuseAndCorruption(t *testing.T) {
	t.Setenv("GOFLAGS", "")
	dir := t.TempDir()
	surfaceFile(t, filepath.Join(dir, "go.mod"), "module example.com/surface\n\ngo 1.27\n")
	cache := t.TempDir()
	first, err := cachedSurface(dir, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 {
		t.Fatal("no overlay roots")
	}
	path := filepath.Join(cache, "surface.json")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	second, err := cachedSurface(dir, cache)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) || len(first) != len(second) {
		t.Fatal("unchanged discovery did not reuse metadata")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved surfaceReceipt
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Packages[0] = nil
	data, err = json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	surfaceFile(t, path, string(data))
	if _, err := cachedSurface(dir, cache); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Packages[0] == nil {
		t.Fatal("corrupt surface receipt survived")
	}
	t.Setenv("GOFLAGS", "-trimpath")
	before := saved.Identity
	if _, err := cachedSurface(dir, cache); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Identity == before {
		t.Fatal("Go environment change reused discovery")
	}
}

func TestSurfaceIdentityTracksSourcesAndFiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOROOT", root)
	t.Setenv("GOTOOLCHAIN", "local")
	dir := t.TempDir()
	surfaceFile(t, filepath.Join(dir, "go.mod"), "module example.com/snapshot\n\ngo 1.27\n")
	file := filepath.Join(root, "src", "fmt", "probe.go")
	surfaceFile(t, file, "package fmt\nconst X=1\n")
	identity := func() string {
		t.Helper()
		key, _, err := surfaceInputs(dir)
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	first := identity()
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	surfaceFile(t, file, "package fmt\nconst X=2\n")
	if err := os.Chtimes(file, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	second := identity()
	if first == second {
		t.Fatal("same-size source edit did not invalidate discovery")
	}
	added := filepath.Join(root, "src", "fmt", "added.go")
	surfaceFile(t, added, "package fmt\n")
	third := identity()
	if second == third {
		t.Fatal("new file did not invalidate discovery")
	}
	if err := os.Remove(added); err != nil {
		t.Fatal(err)
	}
	if identity() != second {
		t.Fatal("source removal did not restore the prior identity")
	}
}
