package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUpstreamSubtestNames(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/testing/match_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "var namingTestCases =")
	end := strings.Index(text[start:], "\nfunc TestNaming(") + start
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/testnames\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "names_test.go"), `// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package testnames
import ("testing";"strings")
`+text[start:end]+`
func TestNames(t *testing.T) {
 for _,tc:=range namingTestCases {
  t.Run(tc.name,func(t *testing.T) {
   if got,want:=strings.TrimPrefix(t.Name(),"TestNames"),strings.TrimPrefix(tc.want,"x");got!=want {t.Errorf("name %q, want %q",got,want)}
  })
 }
}
`)
	want := goTest(t, dir, true)
	got := gartTest(t, dir, driver.TestOptions{Verbose: true})
	if got != want {
		t.Fatalf("subtest names differ:\nGo:\n%s\nGart:\n%s", want, got)
	}
}
