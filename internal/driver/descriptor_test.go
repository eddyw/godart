package driver_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// descVar matches the variable that holds the description of a type with no
// name of its own. Read decision 352. The name of a shared shape file stands in
// front, thus the pattern reads an optional prefix.
var descVar = regexp.MustCompile(`[A-Za-z0-9_$]*\$desc[0-9a-f]{8}`)

// descVarDecl matches the declaration of such a variable.
var descVarDecl = regexp.MustCompile(`(?m)^final TypeDesc ([A-Za-z0-9_$]*\$desc[0-9a-f]{8}) =`)

// methodTypesHead marks the table that holds the function type of each exported
// method of a type.
const methodTypesHead = "methodTypes: <String, TypeDesc Function()>{"

// TestDescriptorVariable proves decision 352. The description of a type with no
// name of its own stands in 1 variable of the library, thus a method type table
// names that variable and holds no description of its own.
//
// The test also proves that each name reaches a declaration of the same library.
// A name with no declaration gives the Dart compiler an unknown symbol.
func TestDescriptorVariable(t *testing.T) {
	for _, prog := range []string{"multi", "generics", "genericiface"} {
		t.Run(prog, func(t *testing.T) {
			source, err := filepath.Abs("../../testdata/programs/" + prog)
			if err != nil {
				t.Fatal(err)
			}
			files := buildOnce(t, source)
			decls := map[string]map[string]bool{}
			uses := map[string]map[string]bool{}
			tables := 0
			for path, text := range files {
				if !strings.HasSuffix(path, ".dart") {
					continue
				}
				lib := libraryOf(path)
				for _, m := range descVarDecl.FindAllStringSubmatch(text, -1) {
					if decls[lib] == nil {
						decls[lib] = map[string]bool{}
					}
					decls[lib][m[1]] = true
				}
				for _, name := range descVar.FindAllString(text, -1) {
					if uses[lib] == nil {
						uses[lib] = map[string]bool{}
					}
					uses[lib][name] = true
				}
				tables += countTables(t, path, text)
			}
			// Only a type with an exported method holds such a table, thus
			// the count of a program with no such type is 0.
			if prog == "multi" && tables == 0 {
				t.Errorf("%s holds no method type table", prog)
			}
			for lib, names := range uses {
				for name := range names {
					if !decls[lib][name] {
						t.Errorf("%s names %s, and the library declares it not", lib, name)
					}
				}
			}
		})
	}
}

// countTables reports the count of the method type tables of the text, and it
// fails when such a table holds a description of its own.
func countTables(t *testing.T, path, text string) int {
	t.Helper()
	count := 0
	rest := text
	for {
		at := strings.Index(rest, methodTypesHead)
		if at < 0 {
			return count
		}
		count++
		body := rest[at+len(methodTypesHead):]
		end := closeIndex(body)
		if end < 0 {
			t.Fatalf("%s holds an open method type table", path)
		}
		if strings.Contains(body[:end], "TypeDesc(") {
			t.Errorf("%s writes a description inside a method type table", path)
		}
		rest = body[end:]
	}
}

// closeIndex gives the offset of the brace that closes the table.
func closeIndex(body string) int {
	depth := 1
	for i, r := range body {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// libraryOf names the library file that holds a part file. The emitter writes
// the part of a package inside the directory of the package.
func libraryOf(path string) string {
	switch {
	case strings.HasSuffix(path, ".part.dart"):
		return filepath.Dir(path) + ".dart"
	case strings.HasSuffix(path, ".map.dart"):
		return strings.TrimSuffix(path, ".map.dart") + ".dart"
	}
	return path
}
