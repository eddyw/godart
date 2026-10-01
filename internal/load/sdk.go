package load

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/version"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type goSDK struct {
	root       string
	buildRoot  string
	fieldTrack bool
	env        []string
}

func prepareGoSDK(dir string) (goSDK, error) {
	cmd := exec.Command("go", "env", "-json", "GOROOT", "GOMODCACHE", "GOVERSION", "GOEXPERIMENT")
	cmd.Dir = dir
	cmd.Env = os.Environ()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return goSDK{}, fmt.Errorf("locate Go SDK: %w: %s", err, stderr.String())
	}
	var selected struct{ GOROOT, GOMODCACHE, GOVERSION, GOEXPERIMENT string }
	if err := json.Unmarshal(data, &selected); err != nil {
		return goSDK{}, err
	}
	sdk := goSDK{root: selected.GOROOT, buildRoot: selected.GOROOT, env: cmd.Env}
	for _, experiment := range strings.Split(selected.GOEXPERIMENT, ",") {
		switch experiment {
		case "fieldtrack":
			sdk.fieldTrack = true
		case "none", "nofieldtrack":
			sdk.fieldTrack = false
		}
	}

	rel, err := filepath.Rel(selected.GOMODCACHE, selected.GOROOT)
	if err != nil || !filepath.IsLocal(rel) {
		return sdk, nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return goSDK{}, err
	}
	sdk.root, err = copyGoSDK(selected.GOROOT, selected.GOVERSION, filepath.Join(cache, "gart", "go-sdk"))
	if err != nil {
		return goSDK{}, err
	}
	sdk.env = nil
	var searchPath string
	for _, item := range cmd.Env {
		key, value, _ := strings.Cut(item, "=")
		if key == "PATH" {
			searchPath = value
		}
		if key != "GOROOT" && key != "GOTOOLCHAIN" && key != "PATH" {
			sdk.env = append(sdk.env, item)
		}
	}
	// GOROOT also selects the copy when the launcher already has this version.
	sdk.env = append(sdk.env,
		"GOROOT="+sdk.root,
		"GOTOOLCHAIN="+selected.GOVERSION,
		"PATH="+filepath.Join(sdk.root, "bin")+string(os.PathListSeparator)+searchPath)
	return sdk, nil
}

// Go forbids overlays below GOMODCACHE. Downloaded SDK modules are immutable,
// thus each source path and version can share an atomic, complete SDK copy.
func copyGoSDK(source, toolchain, cache string) (string, error) {
	if !version.IsValid(toolchain) || filepath.Base(toolchain) != toolchain {
		return "", fmt.Errorf("invalid Go toolchain version: %q", toolchain)
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(source+"\x00"+toolchain)))
	destination := filepath.Join(cache, key)
	if _, err := os.Stat(destination); err == nil {
		return destination, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(cache, 0755); err != nil {
		return "", err
	}
	temporary, err := os.MkdirTemp(cache, ".sdk-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temporary)
	if err := os.CopyFS(temporary, os.DirFS(source)); err != nil {
		return "", fmt.Errorf("copy Go SDK: %w", err)
	}
	if err := os.Link(filepath.Join(temporary, "bin", "go"), filepath.Join(temporary, "bin", toolchain)); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, destination); err != nil {
		if _, present := os.Stat(destination); present != nil {
			return "", err
		}
	}
	return destination, nil
}
