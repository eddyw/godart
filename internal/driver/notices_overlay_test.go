package driver

import (
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestNoticesForSyntheticOverlay(t *testing.T) {
	root := t.TempDir()
	putKernelFile(t, filepath.Join(root, "VERSION"), "go1.27.1\n")
	putKernelFile(t, filepath.Join(root, "LICENSE"), "Go SDK fixture notice\n")
	pkg := &packages.Package{PkgPath: "internal/linkhook", GoFiles: []string{filepath.Join(root, "src/internal/linkhook/linkhook.go")}}
	text, err := sourceNotices([]*packages.Package{pkg})
	if err != nil || !strings.Contains(text, "Go SDK fixture notice") {
		t.Fatalf("notice %q, error %v", text, err)
	}
	pkg.PkgPath = "missing/native"
	if _, err := sourceNotices([]*packages.Package{pkg}); err == nil {
		t.Fatal("accepted a missing native source directory")
	}
}

func TestNoticesForX509Limbo(t *testing.T) {
	pkg := &packages.Package{PkgPath: "crypto/internal/cryptotest/x509limbo"}
	text, err := sourceNotices([]*packages.Package{pkg})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Apache License", "Version 2.0, January 2004", "MIT License", "Copyright (c) 2018 Alexander Staubo", "Permission is hereby granted", "feb7caccc1af"} {
		if !strings.Contains(text, want) {
			t.Errorf("supplemental notice omits %q", want)
		}
	}
	text, err = sourceNotices([]*packages.Package{{PkgPath: "crypto/x509"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "Apache License") {
		t.Fatal("unrelated output includes the test schema license")
	}
}
