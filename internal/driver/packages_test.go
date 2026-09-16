package driver_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type dartPackage struct {
	Name            string `json:"name"`
	RootURI         string `json:"rootUri"`
	PackageURI      string `json:"packageUri"`
	LanguageVersion string `json:"languageVersion"`
}

type dartPackages struct {
	Version  int           `json:"configVersion"`
	Packages []dartPackage `json:"packages"`
}

// Parity fixtures all depend on the same runtime. Resolve that dependency once
// per process, then relocate the generated package without another Pub process.
var fixturePackages = sync.OnceValues(func() (dartPackages, error) {
	dir, err := os.MkdirTemp("", "gart-pub-")
	if err != nil {
		return dartPackages{}, err
	}
	defer os.RemoveAll(dir)
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		return dartPackages{}, err
	}
	manifest := fmt.Sprintf("name: gart_fixture\nenvironment:\n  sdk: ^3.13.0\ndependencies:\n  goruntime:\n    path: %s\n", runtime)
	if err := os.WriteFile(filepath.Join(dir, "pubspec.yaml"), []byte(manifest), 0o600); err != nil {
		return dartPackages{}, err
	}
	cmd := exec.Command("dart", "pub", "get")
	cmd.Dir = dir
	if text, err := cmd.CombinedOutput(); err != nil {
		return dartPackages{}, fmt.Errorf("dart pub get: %w\n%s", err, text)
	}
	path := filepath.Join(dir, ".dart_tool", "package_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return dartPackages{}, err
	}
	var config dartPackages
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	if config.Version != 2 {
		return config, fmt.Errorf("unsupported package configuration version %d", config.Version)
	}
	base := &url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	for i := range config.Packages {
		pkg := &config.Packages[i]
		root, err := url.Parse(pkg.RootURI)
		if err != nil {
			return config, err
		}
		if pkg.Name == "" || pkg.RootURI == "" {
			return config, fmt.Errorf("invalid package configuration: %+v", *pkg)
		}
		pkg.RootURI = base.ResolveReference(root).String()
	}
	return config, nil
})

func prepareDart(t *testing.T, out string) {
	t.Helper()
	start := time.Now()
	config, err := fixturePackages()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(out, "pubspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	name, _, _ := strings.Cut(string(manifest), "\n")
	name = strings.TrimPrefix(name, "name: ")
	config.Packages = append([]dartPackage(nil), config.Packages...)
	for i := range config.Packages {
		if config.Packages[i].Name == "gart_fixture" {
			config.Packages[i].Name = name
			config.Packages[i].RootURI = "../"
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(out, ".dart_tool")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package_config.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("phase=package-config duration=%.3fs", time.Since(start).Seconds())
}

func TestFixturePackageConfiguration(t *testing.T) {
	out := t.TempDir()
	manifest := fmt.Sprintf("name: independent_fixture\nenvironment:\n  sdk: ^3.13.0\ndependencies:\n  goruntime:\n    path: %s\n", runtimeDir(t))
	writeFile(t, filepath.Join(out, "pubspec.yaml"), manifest)
	prepareDart(t, out)
	read := func() map[string]dartPackage {
		t.Helper()
		path := filepath.Join(out, ".dart_tool", "package_config.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var config dartPackages
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatal(err)
		}
		base := &url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
		packages := make(map[string]dartPackage)
		for _, pkg := range config.Packages {
			root, err := url.Parse(pkg.RootURI)
			if err != nil {
				t.Fatal(err)
			}
			pkg.RootURI = strings.TrimSuffix(base.ResolveReference(root).String(), "/")
			packages[pkg.Name] = pkg
		}
		return packages
	}
	got := read()
	run(t, out, "dart", "pub", "get")
	want := read()
	if !maps.Equal(got, want) {
		t.Fatalf("fixture packages differ from Pub:\ngot: %v\nwant: %v", got, want)
	}
}
