package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eddyw/godart/internal/notices"
)

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	root, err := filepath.Abs("../..")
	if err != nil {
		return err
	}
	command := exec.Command("go", "list", "-deps", "-json", "./cmd/gart", "./cmd/glutter")
	command.Dir = root
	data, err := command.Output()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	entries := map[string]string{}
	for {
		var pkg struct {
			Dir    string
			Root   string
			Module *struct{ Path, Version, Dir string }
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		component, sourceRoot := "Go SDK", pkg.Root
		if pkg.Module != nil {
			component = strings.TrimSpace(pkg.Module.Path + " " + pkg.Module.Version)
			sourceRoot = pkg.Module.Dir
		}
		if pkg.Dir == "" || sourceRoot == "" {
			continue
		}
		found, err := notices.Files(pkg.Dir, sourceRoot)
		if err != nil {
			return err
		}
		for name, text := range found {
			if strings.HasPrefix(name, "internal/notices/") {
				continue
			}
			entries[component+" / "+name] = text
		}
	}
	return os.WriteFile("THIRD_PARTY_NOTICES", []byte(notices.Bundle(entries)), 0644)
}
