package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func (a *app) hostInputs() (string, error) {
	config := filepath.Join(a.dir, ".dart_tool", "package_config.json")
	data, err := os.ReadFile(config)
	if err != nil {
		return "", fmt.Errorf("resolve application dependencies with flutter pub get: %w", err)
	}
	var graph struct {
		Version  int `json:"configVersion"`
		Packages []struct {
			RootURI string `json:"rootUri"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &graph); err != nil {
		return "", err
	}
	if graph.Version != 2 || len(graph.Packages) == 0 {
		return "", fmt.Errorf("expected package configuration version 2 with packages: %s", config)
	}
	version, err := exec.Command("dart", "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("dart version: %w", err)
	}
	h := sha256.New()
	h.Write(version)
	fmt.Fprintf(h, "host-inputs-v1\x00%s\x00", data)
	base := &url.URL{Scheme: "file", Path: filepath.ToSlash(config)}
	roots := []string{filepath.Join(a.nodegenPath(), "lib"), filepath.Join(a.nodegenPath(), "bin")}
	files := []string{filepath.Join(a.nodegenPath(), "pubspec.lock"), filepath.Join(a.dir, "pubspec.yaml"), filepath.Join(a.dir, "pubspec.lock")}
	manifest := filepath.Join(a.dir, "glutter.bindings.json")
	if _, err := os.Stat(manifest); err == nil {
		files = append(files, manifest)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	for _, pkg := range graph.Packages {
		if pkg.RootURI == "" {
			return "", fmt.Errorf("package rootUri is empty: %s", config)
		}
		ref, err := url.Parse(pkg.RootURI)
		if err != nil {
			return "", err
		}
		resolved := base.ResolveReference(ref)
		if resolved.Scheme != "file" || resolved.Host != "" || resolved.RawQuery != "" || resolved.Fragment != "" {
			return "", fmt.Errorf("unsupported package root %s", resolved)
		}
		root := filepath.Clean(filepath.FromSlash(resolved.Path))
		if root == filepath.Clean(a.outPath()) {
			continue
		}
		roots = append(roots, filepath.Join(root, "lib"))
		entries, err := os.ReadDir(root)
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			name := strings.ToUpper(entry.Name())
			if !entry.IsDir() && (strings.HasPrefix(name, "LICENSE") || strings.HasPrefix(name, "NOTICE") || strings.HasPrefix(name, "COPYING") || strings.HasPrefix(name, "PATENTS")) {
				files = append(files, filepath.Join(root, entry.Name()))
			}
		}
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if os.IsNotExist(err) && path == root {
				return nil
			}
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".dart") && filepath.Clean(path) != filepath.Clean(a.hostPath()) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	slices.Sort(files)
	files = slices.Compact(files)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", path, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
