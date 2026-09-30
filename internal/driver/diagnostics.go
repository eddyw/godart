package driver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/eddyw/godart/internal/sdk"
)

type Diagnostics struct {
	Go       sdk.GoEnvironment
	Dart     string
	Frontend string
}

func Diagnose(dir, frontend string) (Diagnostics, error) {
	goEnv, err := sdk.ReadGoEnvironment(dir)
	if err != nil {
		return Diagnostics{}, err
	}
	root, err := dartSDK()
	if err != nil {
		return Diagnostics{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, "version"))
	if err != nil {
		return Diagnostics{}, err
	}
	dart := strings.TrimSpace(string(data))
	versions, err := sdk.Read()
	if err != nil {
		return Diagnostics{}, err
	}
	mode, _, err := selectFrontend(frontend, dart, versions)
	if err != nil {
		return Diagnostics{}, err
	}
	return Diagnostics{Go: goEnv, Dart: dart, Frontend: mode}, nil
}
