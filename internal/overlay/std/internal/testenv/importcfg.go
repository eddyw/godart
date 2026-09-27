// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package testenv

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func WriteImportcfg(t testing.TB, dstPath string, packageFiles map[string]string, pkgs ...string) {
	t.Helper()

	icfg := new(bytes.Buffer)
	icfg.WriteString("# import config\n")
	for k, v := range packageFiles {
		fmt.Fprintf(icfg, "packagefile %s=%s\n", k, v)
	}

	if len(pkgs) > 0 {
		// Use 'go list' to resolve any missing packages and rewrite the import map.
		cmd := Command(t, GoToolPath(t), "list", "-export", "-deps", "-f", `{{if ne .ImportPath "command-line-arguments"}}{{if .Export}}{{.ImportPath}}={{.Export}}{{end}}{{end}}`)
		cmd.Args = append(cmd.Args, pkgs...)
		cmd.Stderr = new(strings.Builder)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%v: %v\n%s", cmd, err, cmd.Stderr)
		}

		for line := range strings.SplitSeq(string(out), "\n") {
			if line == "" {
				continue
			}
			importPath, export, ok := strings.Cut(line, "=")
			if !ok {
				t.Fatalf("invalid line in output from %v:\n%s", cmd, line)
			}
			if packageFiles[importPath] == "" {
				fmt.Fprintf(icfg, "packagefile %s=%s\n", importPath, export)
			}
		}
	}

	if err := os.WriteFile(dstPath, icfg.Bytes(), 0666); err != nil {
		t.Fatal(err)
	}
}
