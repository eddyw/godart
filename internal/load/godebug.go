package load

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

func defaultGODEBUG(dir string, patterns, tags []string, tests bool) (map[string]string, error) {
	args := []string{"list", "-e", "-json=ImportPath,Name,DefaultGODEBUG", tagFlag(tags)}
	if tests {
		args = append(args, "-test")
	}
	args = append(args, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Go GODEBUG defaults: %w: %s", err, stderr.String())
	}
	result := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg struct{ ImportPath, Name, DefaultGODEBUG string }
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Go GODEBUG defaults: %w", err)
		}
		if pkg.Name == "main" {
			result[pkg.ImportPath] = pkg.DefaultGODEBUG
		}
	}
	return result, nil
}
