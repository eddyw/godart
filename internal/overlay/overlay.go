// Package overlay holds the Go source that replaces a standard library
// package. The build tag `dart` selects the files, thus the go command skips
// them and the compiler reads them.
//
// The directory std holds the curated surface. The directory stub holds a
// package that the load must replace but the surface does not hold. The go
// command fixes the import list of a test binary, and a stub keeps that list
// out of the type check.
package overlay

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed std stub
var tree embed.FS

// Package holds the overlay of 1 standard library package.
type Package struct {
	// Path is the import path, for example `fmt` or `syscall/dart`.
	Path string
	// Name is the package name.
	Name string
	// Stub marks a package that the load replaces with an empty package. The
	// curated surface does not hold such a package.
	Stub bool
	// Patch marks a package that the load reads from the real Go source. The
	// files of the overlay then replace the file of the same base name, and
	// each other file of the package keeps its content.
	Patch bool
	// Files holds the base name and the content of each file, in the order of
	// the base name.
	Files []File
}

// File is 1 overlay file.
type File struct {
	Name    string
	Content []byte
}

// Packages lists each overlay package, in the order of the import path.
func Packages() []Package {
	out := append(walk("std", false), walk("stub", true)...)
	patch := patchSet()
	for i := range out {
		out[i].Patch = patch[out[i].Path]
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func walk(root string, stub bool) []Package {
	byPath := map[string][]File{}
	err := fs.WalkDir(tree, root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".go") {
			return err
		}
		content, err := tree.ReadFile(name)
		if err != nil {
			return err
		}
		pkgPath := path.Dir(strings.TrimPrefix(name, root+"/"))
		byPath[pkgPath] = append(byPath[pkgPath], File{Name: path.Base(name), Content: content})
		return nil
	})
	if err != nil {
		panic("overlay: " + err.Error())
	}
	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]Package, 0, len(paths))
	for _, p := range paths {
		files := byPath[p]
		sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
		out = append(out, Package{Path: p, Name: path.Base(p), Stub: stub, Files: files})
	}
	return out
}

func patchSet() map[string]bool {
	out := map[string]bool{}
	for _, path := range native {
		out[path] = true
	}
	return out
}

// IntrinsicPath names the package that the compiler replaces with a call to
// the runtime library. The compiler writes no Dart code for it.
const IntrinsicPath = "syscall/dart"

// Intrinsic reports whether the compiler replaces each call of the package.
func Intrinsic(pkgPath string) bool { return pkgPath == IntrinsicPath }

// Paths lists the import path of each package of the curated surface. A stub
// stays out of the list, thus the verifier rejects an import of it and the
// build writes no Dart code for it.
func Paths() []string {
	var out []string
	for _, p := range Packages() {
		if !p.Stub {
			out = append(out, p.Path)
		}
	}
	return out
}
