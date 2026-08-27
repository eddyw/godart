// Package overlay holds the Go source that replaces a standard library
// package. The build tag `dart` selects the files, thus the go command skips
// them and the compiler reads them.
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
	byPath := map[string][]File{}
	err := fs.WalkDir(tree, "std", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".go") {
			return err
		}
		content, err := tree.ReadFile(name)
		if err != nil {
			return err
		}
		pkgPath := path.Dir(strings.TrimPrefix(name, "std/"))
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
		out = append(out, Package{Path: p, Name: path.Base(p), Files: files})
	}
	return out
}

// IntrinsicPath names the package that the compiler replaces with a call to
// the runtime library. The compiler writes no Dart code for it.
const IntrinsicPath = "syscall/dart"

// Intrinsic reports whether the compiler replaces each call of the package.
func Intrinsic(pkgPath string) bool { return pkgPath == IntrinsicPath }

// Paths lists the import path of each overlay package.
func Paths() []string {
	pkgs := Packages()
	out := make([]string, len(pkgs))
	for i, p := range pkgs {
		out[i] = p.Path
	}
	return out
}
