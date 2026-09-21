package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/eddyw/godart/internal/notices"
	"github.com/eddyw/godart/internal/overlay"
)

func sourceNotices(pkgs []*packages.Package) (string, error) {
	entries := map[string]string{"Godart compiler output support": notices.Project}
	visited := map[string]bool{}
	for _, pkg := range pkgs {
		license, err := overlay.License(pkg.PkgPath)
		if err != nil {
			return "", err
		}
		if len(license) != 0 {
			entries["Overlay source / "+pkg.PkgPath] = string(license)
		}

		for _, file := range pkg.Syntax {
			for index, group := range file.Comments {
				if group.Pos() >= file.Package {
					break
				}
				text := group.Text()
				lower := strings.ToLower(text)
				if strings.Contains(lower, "copyright") || strings.Contains(text, "SPDX-License-Identifier:") {
					entries[fmt.Sprintf("Source notice: %s / %s / %d", pkg.PkgPath, filepath.Base(pkg.Fset.Position(file.Pos()).Filename), index)] = text
				}
			}
		}
		for _, file := range pkg.GoFiles {
			dir := filepath.Dir(file)
			if visited[dir] {
				continue
			}
			visited[dir] = true
			root, component := noticeRoot(pkg, dir)
			if root == "" {
				continue
			}
			if _, err := os.Stat(dir); pkg.Module == nil && os.IsNotExist(err) {
				for _, replacement := range overlay.Packages() {
					if replacement.Path != pkg.PkgPath {
						continue
					}
					for dir != root {
						if _, err := os.Stat(dir); !os.IsNotExist(err) {
							break
						}
						dir = filepath.Dir(dir)
					}
					break
				}
			}
			found, err := notices.Files(dir, root)
			if err != nil {
				return "", err
			}
			for name, text := range found {
				entries[component+" / "+name] = text
			}
		}
	}
	return notices.Bundle(entries), nil
}

func noticeRoot(pkg *packages.Package, dir string) (string, string) {
	if pkg.PkgPath == "command-line-arguments" {
		for root := dir; ; root = filepath.Dir(root) {
			if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
				return root, pkg.PkgPath
			}
			if root == filepath.Dir(root) {
				return dir, pkg.PkgPath
			}
		}
	}
	if module := pkg.Module; module != nil {
		importPath := module.Path
		if module.Replace != nil {
			module = module.Replace
		}
		root := module.Dir
		rel, err := filepath.Rel(root, dir)
		if root == "" || err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			root = dir
			for _, part := range strings.Split(strings.TrimPrefix(pkg.PkgPath, importPath), "/") {
				if part != "" {
					root = filepath.Dir(root)
				}
			}
		}
		return root, strings.TrimSpace(importPath + " " + module.Version)
	}
	// Standard packages have no module. Their source can use a selected Go SDK.
	for root := dir; ; root = filepath.Dir(root) {
		if _, err := os.Stat(filepath.Join(root, "VERSION")); err == nil {
			return root, "Go SDK"
		}
		if root == filepath.Dir(root) {
			return "", ""
		}
	}
}
