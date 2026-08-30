// Package meta runs stage 6 of the compiler. It writes 1 archive for each
// compiled package, and it reads the archive again on the next build.
//
// The archive holds 2 hashes. The source hash covers the input of the compile
// function, thus an equal hash means that the package needs no second compile.
// The export hash covers only the exported interface. A change inside a
// function body keeps the export hash equal, thus the dependents keep their
// cache. This mechanism is the early cutoff.
package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Version names the compiler. A new version invalidates each archive.
const Version = "gart-0.1"

// build gives the identity of the compiler binary. The cache key holds it, thus
// a new build of the compiler invalidates each archive.
//
// The version alone is not enough. The compiler changes many times inside 1
// version, and a stale archive then holds the output of the old compiler.
var build = sync.OnceValue(func() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	f, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
})

// Archive holds the metadata of 1 compiled package.
type Archive struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	SourceHash string   `json:"sourceHash"`
	ExportHash string   `json:"exportHash"`
	Export     string   `json:"export"`
	Blocking   []string `json:"blocking"`
	Files      []string `json:"files"`
}

// Store reads and writes the archives of a build.
type Store struct {
	dir string
}

func Open(dir string) *Store { return &Store{dir: dir} }

func (s *Store) path(pkgPath string) string {
	return filepath.Join(s.dir, escape(pkgPath)+".json")
}

// escape maps an import path to a file name. The map is reversible, thus 2
// paths never share a file.
func escape(pkgPath string) string {
	var b strings.Builder
	for _, r := range pkgPath {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			fmt.Fprintf(&b, "%%%02x", r)
		}
	}
	return b.String()
}

func (s *Store) Read(pkgPath string) (*Archive, bool) {
	text, err := os.ReadFile(s.path(pkgPath))
	if err != nil {
		return nil, false
	}
	out := &Archive{}
	if err := json.Unmarshal(text, out); err != nil {
		return nil, false
	}
	return out, true
}

func (s *Store) Write(a *Archive) error {
	if err := os.MkdirAll(s.dir, 0o777); err != nil {
		return err
	}
	text, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(a.Path), append(text, '\n'), 0o666)
}

// SourceHash covers each input of the compile function: the source files, the
// export hash of each dependency, the identity of the compiler, the flags, and
// the digest of the block analysis.
func SourceHash(files map[string][]byte, deps []string, flags, blocks string) string {
	h := sha256.New()
	fmt.Fprintf(h, "version\x00%s\x00%s\x00", Version, build())
	fmt.Fprintf(h, "flags\x00%s\x00", flags)
	fmt.Fprintf(h, "blocks\x00%s\x00", blocks)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(h, "file\x00%s\x00%d\x00", filepath.Base(name), len(files[name]))
		h.Write(files[name])
	}
	sorted := append([]string{}, deps...)
	sort.Strings(sorted)
	for _, d := range sorted {
		fmt.Fprintf(h, "dep\x00%s\x00", d)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ExportHash covers only the exported interface. The block state of an
// exported function belongs to that interface, because a change from
// synchronous to asynchronous changes each call site.
func ExportHash(export []byte, blocking []string) string {
	h := sha256.New()
	fmt.Fprintf(h, "export\x00%d\x00", len(export))
	h.Write(export)
	sorted := append([]string{}, blocking...)
	sort.Strings(sorted)
	for _, name := range sorted {
		fmt.Fprintf(h, "blocks\x00%s\x00", name)
	}
	return hex.EncodeToString(h.Sum(nil))
}
