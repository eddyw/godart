package load

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// The compiler holds no directive of the linker, thus a `//go:embed` variable
// takes its value from the source.
//
// The go command rejects a directive on a variable that holds an initializer.
// Thus the rewrite turns each directive into a plain comment, and it gives the
// variable the value of the files that the patterns name. The initialization
// order of Go then holds, and a package level variable that reads the value
// sees it.
//
// The second load parses the rewritten source without a Go command overlay.
// This permits embedded files in downloaded modules beneath GOMODCACHE.

// embedMark is the text that replaces the directive. The go command reads a
// directive at the start of a comment alone, thus the space is enough.
const embedMark = "// go:embed"

// embedRewrites writes the source of each file of the module that holds a
// `//go:embed` directive into files. It gives the path of each such file.
func embedRewrites(found []*packages.Package, files map[string][]byte) (map[string]bool, error) {
	var pkgs []*packages.Package
	packages.Visit(found, nil, func(p *packages.Package) {
		// A native package of the standard library also holds a directive.
		// `encoding/json/internal/jsontest` embeds the corpus of the real test
		// files of `encoding/json`. The loop below skips each file that the
		// overlay already replaced.
		pkgs = append(pkgs, p)
	})
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ID < pkgs[j].ID })

	out := map[string]bool{}
	for _, p := range pkgs {
		for _, name := range p.CompiledGoFiles {
			if out[name] {
				continue
			}
			if _, held := files[name]; held {
				continue
			}
			src, err := os.ReadFile(name)
			if err != nil {
				return nil, err
			}
			if !bytes.Contains(src, []byte("//go:embed")) {
				continue
			}
			text, err := rewriteEmbed(name, src)
			if err != nil {
				return nil, err
			}
			if text == nil {
				continue
			}
			files[name] = text
			out[name] = true
		}
	}
	return out, nil
}

// embedKind names the 3 types that a directive accepts.
type embedKind int

const (
	embedNone embedKind = iota
	embedString
	embedBytes
	embedFS
)

// rewriteEmbed gives the source of 1 file with each directive replaced by the
// value of its files. It gives nil when the file holds no directive.
func rewriteEmbed(name string, src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(name)
	pkgName := embedImportName(file)
	base := fset.File(file.Pos()).Base()

	// An edit holds a text and the offset that takes it. The rewrite applies
	// each edit from the end of the file, thus an earlier offset stays valid.
	type edit struct {
		off  int
		drop int
		text string
	}
	var edits []edit
	var localErr error
	ast.Inspect(file, func(n ast.Node) bool {
		decl, ok := n.(*ast.DeclStmt)
		if !ok {
			return true
		}
		gen, ok := decl.Decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			return true
		}
		groups := []*ast.CommentGroup{gen.Doc}
		for _, spec := range gen.Specs {
			groups = append(groups, spec.(*ast.ValueSpec).Doc)
		}
		for _, group := range groups {
			for _, comment := range embedComments(group) {
				localErr = errors.Join(localErr, fmt.Errorf("%s: go:embed cannot apply to var inside func", fset.Position(comment.Pos())))
			}
		}
		return true
	})
	if localErr != nil {
		return nil, localErr
	}

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			doc := vs.Doc
			if doc == nil && len(gen.Specs) == 1 {
				doc = gen.Doc
			}
			lines := embedComments(doc)
			if len(lines) == 0 {
				continue
			}
			for _, c := range lines {
				edits = append(edits, edit{off: int(c.Pos()) - base, drop: 2, text: embedMark})
			}
			patterns, err := embedPatterns(lines)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", fset.Position(vs.Pos()), err)
			}
			kind := embedTypeOf(vs.Type, pkgName)
			if kind == embedNone {
				return nil, fmt.Errorf("%s: a go:embed variable holds the type string, []byte, or %s.FS",
					fset.Position(vs.Pos()), pkgName)
			}
			if len(vs.Names) != 1 || len(vs.Values) != 0 {
				return nil, fmt.Errorf("%s: a go:embed variable takes 1 name and no value",
					fset.Position(vs.Pos()))
			}
			held, err := resolveEmbed(dir, patterns)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", fset.Position(vs.Pos()), err)
			}
			value, err := embedValue(kind, held, pkgName)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", fset.Position(vs.Pos()), err)
			}
			edits = append(edits, edit{off: int(vs.End()) - base, text: " = " + value})
		}
	}
	if len(edits) == 0 {
		return nil, nil
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].off > edits[j].off })
	out := append([]byte{}, src...)
	for _, e := range edits {
		out = append(out[:e.off:e.off], append([]byte(e.text), out[e.off+e.drop:]...)...)
	}
	return out, nil
}

// embedImportName gives the local name of the import of the package embed. It
// gives "embed" when the file imports the package for its directive alone.
func embedImportName(file *ast.File) string {
	for _, imp := range file.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil || p != "embed" {
			continue
		}
		if imp.Name != nil && imp.Name.Name != "_" {
			return imp.Name.Name
		}
		return "embed"
	}
	return "embed"
}

// embedComments gives each `//go:embed` line of a doc comment.
func embedComments(doc *ast.CommentGroup) []*ast.Comment {
	if doc == nil {
		return nil
	}
	var out []*ast.Comment
	for _, c := range doc.List {
		if strings.HasPrefix(c.Text, "//go:embed") {
			out = append(out, c)
		}
	}
	return out
}

// embedPatterns gives the patterns of each directive line. A pattern that
// holds a space stands in quotes, as the go command demands.
func embedPatterns(lines []*ast.Comment) ([]string, error) {
	var out []string
	for _, c := range lines {
		rest := strings.TrimPrefix(c.Text, "//go:embed")
		fields, err := embedFields(rest)
		if err != nil {
			return nil, err
		}
		if len(fields) == 0 {
			return nil, fmt.Errorf("the directive %q names no pattern", c.Text)
		}
		out = append(out, fields...)
	}
	return out, nil
}

// embedFields splits the text of a directive. A field stands alone, or inside
// a pair of double quotes, or inside a pair of back quotes.
func embedFields(text string) ([]string, error) {
	var out []string
	for text = strings.TrimSpace(text); text != ""; text = strings.TrimSpace(text) {
		var field string
		switch text[0] {
		case '"', '`':
			end := strings.IndexByte(text[1:], text[0])
			if end < 0 {
				return nil, fmt.Errorf("the directive holds no closing quote")
			}
			quoted := text[:end+2]
			text = text[end+2:]
			var err error
			field, err = strconv.Unquote(quoted)
			if err != nil {
				return nil, fmt.Errorf("the pattern %s holds no valid quote", quoted)
			}
		default:
			end := strings.IndexAny(text, " \t")
			if end < 0 {
				end = len(text)
			}
			field, text = text[:end], text[end:]
		}
		out = append(out, field)
	}
	return out, nil
}

// embedTypeOf reads the type of the variable of a directive.
func embedTypeOf(t ast.Expr, pkgName string) embedKind {
	switch t := t.(type) {
	case *ast.Ident:
		if t.Name == "string" {
			return embedString
		}
	case *ast.ArrayType:
		if t.Len != nil {
			return embedNone
		}
		if elem, ok := t.Elt.(*ast.Ident); ok && elem.Name == "byte" {
			return embedBytes
		}
	case *ast.SelectorExpr:
		x, ok := t.X.(*ast.Ident)
		if ok && x.Name == pkgName && t.Sel.Name == "FS" {
			return embedFS
		}
	}
	return embedNone
}

// embedFile holds 1 file of a directive.
type embedFile struct {
	// name is the path of the file, with slash separators, relative to the
	// directory of the package.
	name string
	data []byte
}

// resolveEmbed gives each file that the patterns name.
//
// A pattern that names a directory takes each file under it. A name that
// starts with a dot or with a low line stays out, and the prefix `all:` takes
// it. A pattern that names no file gives an error, as the go command does.
func resolveEmbed(dir string, patterns []string) ([]embedFile, error) {
	held := map[string]bool{}
	var out []embedFile
	add := func(full, rel string) error {
		if held[rel] {
			return nil
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		held[rel] = true
		out = append(out, embedFile{name: rel, data: data})
		return nil
	}
	for _, pattern := range patterns {
		all := strings.HasPrefix(pattern, "all:")
		glob := strings.TrimPrefix(pattern, "all:")
		if err := validEmbedPattern(glob); err != nil {
			return nil, fmt.Errorf("the pattern %s is outside the rules: %w", pattern, err)
		}
		matches, err := filepath.Glob(filepath.Join(dir, filepath.FromSlash(glob)))
		if err != nil {
			return nil, fmt.Errorf("the pattern %s holds no valid glob: %w", pattern, err)
		}
		sort.Strings(matches)
		count := 0
		for _, match := range matches {
			info, err := os.Lstat(match)
			if err != nil {
				return nil, err
			}
			rel, err := relSlash(dir, match)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				n, err := walkEmbedDir(dir, match, all, add)
				if err != nil {
					return nil, err
				}
				count += n
				continue
			}
			if !info.Mode().IsRegular() {
				continue
			}
			if elem := vcsName(rel); elem != "" {
				return nil, fmt.Errorf("the pattern %s names the file %s, and no module holds the directory %s of a version control system",
					pattern, rel, elem)
			}
			if err := add(match, rel); err != nil {
				return nil, err
			}
			count++
		}
		if count == 0 {
			return nil, fmt.Errorf("the pattern %s names no file", pattern)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

// walkEmbedDir adds each file under root, and it gives the count.
func walkEmbedDir(dir, root string, all bool, add func(full, rel string) error) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != root && !all && hiddenName(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, err := relSlash(dir, p)
		if err != nil {
			return err
		}
		if err := add(p, rel); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

// hiddenName reports whether the name starts with a dot or with a low line.
// The go command keeps such a file out of a directory pattern.
func hiddenName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// vcsName gives the first element of the path that names the directory of a
// version control system, and it gives an empty text for each other path. No
// module holds such a directory, thus the go command embeds no file under it.
//
// The rule of the dot and of the low line reaches a directory walk alone. A
// pattern that names a file under `_embed` thus embeds it, and
// `encoding/json/internal/jsontest` of Go holds exactly that directive.
func vcsName(rel string) string {
	for _, elem := range strings.Split(rel, "/") {
		switch elem {
		case ".bzr", ".hg", ".git", ".svn":
			return elem
		}
	}
	return ""
}

// validEmbedPattern reports whether the pattern stays inside the directory of
// the package.
func validEmbedPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("the pattern is empty")
	}
	if filepath.IsAbs(pattern) || strings.HasPrefix(pattern, "/") {
		return fmt.Errorf("the pattern is absolute")
	}
	if !fs.ValidPath(pattern) {
		return fmt.Errorf("the pattern holds an element that is empty, a dot, or 2 dots")
	}
	return nil
}

// relSlash gives the path of full relative to dir, with slash separators.
func relSlash(dir, full string) (string, error) {
	rel, err := filepath.Rel(dir, full)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("the file %s stands outside the directory of the package", rel)
	}
	return rel, nil
}

// embedValue gives the Go expression that holds the files.
func embedValue(kind embedKind, held []embedFile, pkgName string) (string, error) {
	switch kind {
	case embedString, embedBytes:
		if len(held) != 1 {
			return "", fmt.Errorf("a go:embed variable of the type string or []byte takes 1 file, and the patterns name %d",
				len(held))
		}
		text := strconv.Quote(string(held[0].data))
		if kind == embedBytes {
			return "[]byte(" + text + ")", nil
		}
		return text, nil
	case embedFS:
		var b strings.Builder
		fmt.Fprintf(&b, "%s.New([]string{", pkgName)
		for _, f := range held {
			fmt.Fprintf(&b, "\n\t%s,", strconv.Quote(f.name))
		}
		b.WriteString("\n}, []string{")
		for _, f := range held {
			fmt.Fprintf(&b, "\n\t%s,", strconv.Quote(string(f.data)))
		}
		b.WriteString("\n})")
		return b.String(), nil
	}
	return "", fmt.Errorf("the type of the variable holds no embed form")
}

// EmbedPath reports whether the path names a file that the loader rewrote for
// a `//go:embed` directive. Stage 2 allows the constructor of the package
// embed in such a file alone.
func (r *Result) EmbedPath(name string) bool { return r.Embeds[name] }
