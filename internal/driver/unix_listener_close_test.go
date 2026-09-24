package driver_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUnixListenerCloseOwnership(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("this adapter covers Linux and macOS Unix sockets")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unixunlink\n\ngo 1.27\n")
	root := filepath.Join(runtime.GOROOT(), "src/net")
	source := selectedUpstreamSource(t, filepath.Join(root, "unixsock_test.go"), []string{"TestUnixUnlink"})
	source = strings.Replace(source, "package net\n", "package unixunlink\nimport . \"net\"\n", 1)
	helper := selectedUpstreamSource(t, filepath.Join(root, "mockserver_test.go"), []string{"testUnixAddr"})
	helper = strings.Replace(helper, "package net\n", "package unixunlink\n", 1)
	helper += "\nfunc testableNetwork(network string) bool { return network == \"unix\" }\n"
	writeFile(t, filepath.Join(dir, "address_test.go"), helper)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "ownership_test.go"), `package unixunlink
import ("net"; "os"; "syscall"; "testing"; "time")
func TestClosePreservesReplacement(t *testing.T) {
 path := testUnixAddr(t)
 for i := 0; i < 1000; i++ {
 listener, err := net.Listen("unix", path)
 if err != nil { t.Fatal(err) }
 if err := listener.Close(); err != nil { t.Fatal(err) }
 fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_EXCL|syscall.O_RDWR, 0600)
 if err != nil { t.Fatal(err) }
 if err := syscall.Close(fd); err != nil { t.Fatal(err) }
 time.Sleep(time.Millisecond)
 if _, err := os.Stat(path); err != nil { t.Fatalf("iteration %d: listener close removed its replacement: %v", i, err) }
 if err := os.Remove(path); err != nil { t.Fatal(err) }
 }
}
`)

	t.Log(run(t, dir, "go", "test", "-v", "."))

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "upstream_test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var omitted []ast.Stmt
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "TestUnixUnlink" {
			continue
		}
		for _, stmt := range fn.Body.List {
			switch stmt := stmt.(type) {
			case *ast.AssignStmt:
				if name, ok := stmt.Lhs[0].(*ast.Ident); ok && name.Name == "fileListener" {
					omitted = append(omitted, stmt)
				}
			case *ast.ExprStmt:
				call, ok := stmt.X.(*ast.CallExpr)
				if !ok || len(call.Args) != 2 {
					continue
				}
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				name, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				if strings.HasPrefix(name, "FileListener") || strings.Contains(name, "SetUnlinkOnClose") {
					t.Logf("unobserved Gart subtest %s: descriptor adoption or unlink control is unavailable", name)
					omitted = append(omitted, stmt)
				}
			}
		}
	}
	if len(omitted) != 6 {
		t.Fatal("the upstream descriptor helper or subtests changed")
	}
	for i := len(omitted) - 1; i >= 0; i-- {
		stmt := omitted[i]
		start, end := fset.Position(stmt.Pos()).Offset, fset.Position(stmt.End()).Offset
		source = source[:start] + strings.Repeat("\n", strings.Count(source[start:end], "\n")) + source[end:]
	}
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			if target == "vm" {
				t.Log(run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v"))
			} else {
				result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v"})
				t.Log(result.Text)
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
