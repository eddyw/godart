package load

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eddyw/godart/internal/overlay"
	"golang.org/x/tools/go/packages"
)

type surfaceReceipt struct {
	Identity string
	Hash     string
	Extra    map[string]string
	Packages []*packages.Package
}

// Only native standard-library roots enter this cache, independent of module layout.
func surfaceInputs(dir string, tags ...string) (string, string, error) {
	cmd := exec.Command("go", "env", "-json", "GOROOT", "GOOS", "GOARCH", "GOAMD64", "GOARM", "GOARM64", "GO386", "GOMIPS", "GOMIPS64", "GOPPC64", "GORISCV64", "GOWASM", "GOEXPERIMENT", "CGO_ENABLED", "GOFLAGS", "GOVERSION", "GOTOOLCHAIN", "GOCACHE", "GOENV", "CC", "CXX", "CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS", "PKG_CONFIG")
	cmd.Dir = dir
	environment, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	var env struct {
		GOROOT  string
		GOFLAGS string
	}
	if err := json.Unmarshal(environment, &env); err != nil {
		return "", "", err
	}
	// An external overlay can replace the source outside GOROOT.
	if strings.Contains(env.GOFLAGS, "-overlay") || os.Getenv("GOPACKAGESDRIVER") != "" {
		return "", "", nil
	}
	sources := map[string]string{}
	for _, p := range overlay.Packages() {
		if p.Path == overlay.IntrinsicPath {
			continue
		}
		path := filepath.Join(env.GOROOT, "src", filepath.FromSlash(p.Path))
		entries, err := os.ReadDir(path)
		if os.IsNotExist(err) {
			sources[path] = "absent"
			continue
		}
		if err != nil {
			return "", "", err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") || strings.HasPrefix(entry.Name(), "_") || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			name := filepath.Join(path, entry.Name())
			data, err := os.ReadFile(name)
			if err != nil {
				return "", "", err
			}
			sum := sha256.Sum256(data)
			sources[name] = hex.EncodeToString(sum[:])
		}
	}
	data, err := json.Marshal(struct {
		Version     string
		Tags        string
		Environment json.RawMessage
		Sources     map[string]string
	}{BuildTag + "-surface-3", tagFlag(tags), environment, sources})
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), env.GOROOT, nil
}

var surfaceMemory struct {
	sync.Mutex
	receipt surfaceReceipt
}

func validSurface(receipt surfaceReceipt, identity string) bool {
	if receipt.Identity != identity || len(receipt.Packages) == 0 {
		return false
	}
	for path, want := range receipt.Extra {
		content, err := os.ReadFile(path)
		sum := sha256.Sum256(content)
		if err != nil || hex.EncodeToString(sum[:]) != want {
			return false
		}
	}
	for _, p := range receipt.Packages {
		if p == nil || p.ID == "" {
			return false
		}
	}
	encoded, _ := json.Marshal(receipt.Packages)
	sum := sha256.Sum256(encoded)
	return receipt.Hash == hex.EncodeToString(sum[:])
}

func cachedSurface(dir, cacheDir string, tags ...string) ([]*packages.Package, error) {
	identity, goroot, err := surfaceInputs(dir, tags...)
	if err != nil {
		return nil, err
	}
	if identity == "" {
		return discoverSurface(dir, tags...)
	}
	path := filepath.Join(cacheDir, "surface.json")
	data, err := os.ReadFile(path)
	var receipt surfaceReceipt
	if err == nil && json.Unmarshal(data, &receipt) == nil && validSurface(receipt, identity) {
		return receipt.Packages, nil
	}
	surfaceMemory.Lock()
	memory := surfaceMemory.receipt
	surfaceMemory.Unlock()
	if validSurface(memory, identity) {
		if err := writeSurface(cacheDir, memory); err != nil {
			return nil, err
		}
		return memory.Packages, nil
	}
	found, err := discoverSurface(dir, tags...)
	if err != nil {
		return nil, err
	}
	after, _, err := surfaceInputs(dir, tags...)
	if err != nil {
		return nil, err
	}
	if after != identity {
		return found, nil
	}
	// Synthetic overlays need no native file list. All native overlay roots occur explicitly.
	roots := make([]*packages.Package, 0, len(found))
	extra := map[string]string{}
	for _, p := range found {
		if len(p.GoFiles) == 0 {
			if _, err := os.Stat(filepath.Join(goroot, "src", filepath.FromSlash(p.PkgPath))); os.IsNotExist(err) {
				continue
			}
		}
		if len(p.Errors) > 0 {
			return found, nil
		}
		for _, file := range p.GoFiles {
			relative, err := filepath.Rel(filepath.Join(goroot, "src"), file)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return found, nil
			}
		}
		for _, file := range p.CompiledGoFiles {
			relative, err := filepath.Rel(filepath.Join(goroot, "src"), file)
			if err != nil {
				return nil, err
			}
			if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				content, err := os.ReadFile(file)
				if err != nil {
					return found, nil
				}
				sum := sha256.Sum256(content)
				extra[file] = hex.EncodeToString(sum[:])
			}
		}
		copy := *p
		copy.Imports = nil
		roots = append(roots, &copy)
	}
	encoded, err := json.Marshal(roots)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(encoded)
	receipt = surfaceReceipt{Identity: identity, Hash: hex.EncodeToString(sum[:]), Extra: extra, Packages: roots}
	surfaceMemory.Lock()
	surfaceMemory.receipt = receipt
	surfaceMemory.Unlock()
	if err := writeSurface(cacheDir, receipt); err != nil {
		return nil, err
	}
	return roots, nil
}

func writeSurface(cacheDir string, receipt surfaceReceipt) error {
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(cacheDir, "surface-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), filepath.Join(cacheDir, "surface.json"))
}

func discoverSurface(dir string, tags ...string) ([]*packages.Package, error) {
	var paths []string
	for _, o := range overlay.Packages() {
		if o.Path != overlay.IntrinsicPath {
			paths = append(paths, o.Path)
		}
	}
	return packages.Load(&packages.Config{Mode: findMode, Dir: dir, BuildFlags: []string{tagFlag(tags)}}, paths...)
}
