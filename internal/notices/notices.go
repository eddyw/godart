package notices

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

//go:generate go run ./generate

//go:embed THIRD_PARTY_NOTICES
var Compiler string

//go:embed LICENSE
var Project string

// Files reads notices from dir through root, including package-specific terms.
func Files(dir, root string) (map[string]string, error) {
	dir = filepath.Clean(dir)
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("notice directory %q is outside %q", dir, root)
	}
	out := map[string]string{}
	for {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !isNotice(entry.Name()) {
				continue
			}
			name := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(name)
			if err != nil {
				return nil, err
			}
			rel, err := filepath.Rel(root, name)
			if err != nil {
				return nil, err
			}
			out[filepath.ToSlash(rel)] = string(data)
		}
		if dir == root {
			return out, nil
		}
		dir = filepath.Dir(dir)
	}
}

func isNotice(name string) bool {
	name = strings.ToUpper(name)
	for _, prefix := range []string{"LICENSE", "LICENCE", "COPYING", "NOTICE", "PATENTS", "THIRD_PARTY_NOTICES"} {
		if name == prefix || strings.HasPrefix(name, prefix+".") || strings.HasPrefix(name, prefix+"-") {
			return true
		}
	}
	return false
}

// Bundle uses Flutter's multi-license format. Each key identifies its component.
func Bundle(entries map[string]string) string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, key+"\n\n"+strings.TrimRight(entries[key], "\n")+"\n")
	}
	return strings.Join(parts, "\n--------------------------------------------------------------------------------\n\n")
}
