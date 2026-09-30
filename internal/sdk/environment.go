package sdk

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
)

type GoEnvironment struct {
	Version       string
	Root          string
	Module        string
	ModuleVersion string
}

func ReadGoEnvironment(dir string) (GoEnvironment, error) {
	cmd := exec.Command("go", "env", "-json", "GOVERSION", "GOROOT", "GOMOD")
	cmd.Dir = dir
	data, err := cmd.Output()
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			return GoEnvironment{}, fmt.Errorf("go env: %w: %s", err, strings.TrimSpace(string(failure.Stderr)))
		}
		return GoEnvironment{}, fmt.Errorf("go env: %w", err)
	}
	var selected struct{ GOVERSION, GOROOT, GOMOD string }
	if err := json.Unmarshal(data, &selected); err != nil {
		return GoEnvironment{}, fmt.Errorf("go env: %w", err)
	}
	if selected.GOVERSION == "" || selected.GOROOT == "" {
		return GoEnvironment{}, fmt.Errorf("go env omitted GOVERSION or GOROOT")
	}
	out := GoEnvironment{Version: selected.GOVERSION, Root: selected.GOROOT}
	if selected.GOMOD == "" || selected.GOMOD == os.DevNull {
		return out, nil
	}
	data, err = os.ReadFile(selected.GOMOD)
	if err != nil {
		return GoEnvironment{}, err
	}
	module, err := modfile.Parse(selected.GOMOD, data, nil)
	if err != nil {
		return GoEnvironment{}, err
	}
	if module.Module == nil {
		return GoEnvironment{}, fmt.Errorf("%s has no module directive", selected.GOMOD)
	}
	out.Module = module.Module.Mod.Path
	if module.Go != nil {
		out.ModuleVersion = module.Go.Version
	}
	return out, nil
}
