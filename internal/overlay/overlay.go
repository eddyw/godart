// Package overlay holds the Go source that replaces a standard library
// package. The build tag `dart` selects the files, thus the go command skips
// them and the compiler reads them.
//
// The directory std holds the curated surface.
package overlay

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed std
var tree embed.FS

// Package holds the overlay of 1 standard library package.
type Package struct {
	// Path is the import path, for example `fmt` or `syscall/dart`.
	Path string
	// Name is the package name.
	Name string
	// Patch marks a package that the load reads from the real Go source. The
	// files of the overlay then replace the file of the same base name, and
	// each other file of the package keeps its content.
	Patch bool
	// Files holds the base name and the content of each file, in the order of
	// the base name.
	Files []File
	// Patches holds each unified diff of the directory, in the order of the
	// base name. A diff `<base>.patch` changes the file `<base>` of the
	// package, and 2 kinds of file take one.
	//
	// A test file of the package under test is the first kind. Such a file
	// stays outside the overlay, and a copy of it would age with each release
	// of Go.
	//
	// A source file of a native package is the second kind. The compiler reads
	// such a package from the real Go source, thus a change of 1 line needs no
	// copy of the whole file. A native package that holds a diff of each of
	// its changes carries no file of its own.
	Patches []File
}

// File is 1 overlay file.
type File struct {
	Name    string
	Content []byte
}

// Packages lists each overlay package, in the order of the import path.
func Packages() []Package {
	out := walk("std")
	patch := patchSet()
	for i := range out {
		out[i].Patch = patch[out[i].Path]
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func walk(root string) []Package {
	byPath := map[string][]File{}
	patches := map[string][]File{}
	err := fs.WalkDir(tree, root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		into := byPath
		if strings.HasSuffix(name, ".patch") {
			into = patches
		} else if !strings.HasSuffix(name, ".go") {
			return nil
		}
		content, err := tree.ReadFile(name)
		if err != nil {
			return err
		}
		pkgPath := path.Dir(strings.TrimPrefix(name, root+"/"))
		into[pkgPath] = append(into[pkgPath], File{Name: path.Base(name), Content: content})
		return nil
	})
	if err != nil {
		panic("overlay: " + err.Error())
	}
	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	for p := range patches {
		if byPath[p] == nil {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	out := make([]Package, 0, len(paths))
	for _, p := range paths {
		files := byPath[p]
		sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
		diffs := patches[p]
		sort.Slice(diffs, func(i, j int) bool { return diffs[i].Name < diffs[j].Name })
		out = append(out, Package{Path: p, Name: path.Base(p), Files: files, Patches: diffs})
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

// Paths lists the import path of each package of the curated surface.
func Paths() []string {
	out := make([]string, 0, len(Packages()))
	for _, p := range Packages() {
		out = append(out, p.Path)
	}
	return out
}
