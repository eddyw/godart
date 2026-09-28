package driver_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestHTTPUpstream(t *testing.T) {
	dir := t.TempDir()
	names := map[string]bool{"reqFunc": true, "h12Compare": true, "slurpResult": true, "mostlyCopy": true,
		"TestTransportHeadResponses": true, "testTransportHeadResponses": true,
		"TestClientRedirectsContext": true, "testClientRedirectsContext": true,
		"TestClientRedirectNoLocation": true, "testClientRedirectNoLocation": true,
		"TestH12_AutoGzip": true, "TestH12_AutoGzip_Disabled": true,
		"TestRoundTripGzip": true, "testRoundTripGzip": true, "roundTripTests": true,
		"TestTransportGzip": true, "testTransportGzip": true,
		"TestTransportRangeAndGzip": true, "testTransportRangeAndGzip": true,
		"TestTransportGzipShort": true, "testTransportGzipShort": true,
		"TestTransportReadToEndReusesConn": true, "testTransportReadToEndReusesConn": true,
		"TestContentEncodingNoSniffing": true, "testContentEncodingNoSniffing": true,
		"TestServerContentTypeSniff": true, "testServerContentTypeSniff": true, "sniffTests": true,
		"TestServerIssue5953": true, "testServerIssue5953": true,
		"TestSniffWriteSize": true, "testSniffWriteSize": true,
		"TestIdentityResponse": true, "testIdentityResponse": true,
		"TestH12_ExplicitContentLength": true, "TestH12_FlushBeforeBody": true, "TestH12_FlushMidBody": true}
	var tests []string
	for name := range names {
		if strings.HasPrefix(name, "Test") {
			tests = append(tests, name)
		}
	}
	slices.Sort(tests)
	var extracted strings.Builder
	extracted.WriteString("// Copyright 2015 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\npackage http_test\n\nimport (\"net\";\"bytes\";\"compress/gzip\";\"compress/zlib\";\"log\";\"slices\";\"context\";\"errors\";\"time\";\"net/url\";\"crypto/rand\";\"fmt\";\"io\";\"reflect\";\"strconv\";\"strings\";\"testing\";. \"net/http\")\n\n")
	for _, file := range []string{"clientserver_test.go", "transport_test.go", "client_test.go", "serve_test.go", "sniff_test.go"} {
		source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/net/http", file))
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			keep := false
			switch d := decl.(type) {
			case *ast.FuncDecl:
				keep = names[d.Name.Name]
				if d.Recv != nil {
					switch typ := d.Recv.List[0].Type.(type) {
					case *ast.Ident:
						keep = names[typ.Name]
					case *ast.StarExpr:
						if n, ok := typ.X.(*ast.Ident); ok {
							keep = names[n.Name]
						}
					}
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch x := spec.(type) {
					case *ast.TypeSpec:
						keep = names[x.Name.Name]
					case *ast.ValueSpec:
						for _, n := range x.Names {
							if names[n.Name] {
								keep = true
							}
						}
					}
				}
			}
			if keep {
				extracted.Write(source[fset.Position(decl.Pos()).Offset:fset.Position(decl.End()).Offset])
				extracted.WriteString("\n\n")
			}
		}
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/httpupstream\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "upstream_test.go"), extracted.String())
	adapter, err := os.ReadFile("testdata/http-upstream-adapter.txt")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "adapter_test.go"), string(adapter))
	patterns := make([]string, len(tests))
	for i, name := range tests {
		patterns[i] = regexp.QuoteMeta(name)
	}
	filter := "^(" + strings.Join(patterns, "|") + ")$/^(h1|h2)$"
	outDir := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: outDir, Runtime: runtimeDir(t), TrimPath: true, Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d HTTP test binaries", len(stats.Binaries))
	}
	prepareDart(t, outDir)
	args := append([]string{filepath.Join(outDir, stats.Binaries[0].Entry)}, upstreamTestArgs("-test.run="+filter)...)
	out, runErr, finished := output(dir, nil, runLimit, "dart", args...)
	code := 0
	if runErr != nil {
		code = 1
	}
	if !finished {
		t.Fatalf("HTTP upstream timeout:\n%s", out)
	}
	compareUpstream(t, "net/http-selected", "net/http", filter, "vm", filepath.Join(runtime.GOROOT(), "src/net/http"), string(out), code)
}
