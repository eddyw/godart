package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	compatibility "github.com/eddyw/godart/internal/sdk"
	"golang.org/x/mod/semver"
)

func selectFrontend(mode, version string, versions compatibility.Versions) (string, bool, error) {
	if mode == "" {
		mode = "auto"
	}
	if mode != "auto" && mode != "resident" && mode != "source" {
		return "", false, fmt.Errorf("unknown frontend %q: use auto, resident, or source", mode)
	}
	if !semver.IsValid("v"+version) || semver.Compare("v"+version, "v"+versions.Dart.SourceMin) < 0 || semver.Compare("v"+version, "v"+versions.Dart.SourceMaxExclusive) >= 0 {
		return "", false, fmt.Errorf("Dart %s is outside the source SDK range >=%s <%s", version, versions.Dart.SourceMin, versions.Dart.SourceMaxExclusive)
	}
	if mode == "source" {
		return "source", false, nil
	}
	if slices.Contains(versions.Dart.Resident, version) {
		return "resident", false, nil
	}
	if mode == "resident" {
		return "", false, fmt.Errorf("resident compilation requires Dart %s, found %s", strings.Join(versions.Dart.Resident, " or "), version)
	}
	return "source", true, nil
}

func (opt *Options) selectExecutionFrontend() error {
	root, err := dartSDK()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, "version"))
	if err != nil {
		return err
	}
	versions, err := compatibility.Read()
	if err != nil {
		return err
	}
	version := strings.TrimSpace(string(data))
	mode, warning, err := selectFrontend(opt.Frontend, version, versions)
	if err != nil {
		return err
	}
	opt.Frontend = mode
	if warning {
		fmt.Fprintf(os.Stderr, "gart: Dart %s has no verified resident frontend; use dart run for this invocation (-frontend=source suppresses this warning)\n", version)
	}
	return nil
}

func (opt Options) executionEntry(entry string) (string, error) {
	if opt.Frontend == "source" {
		return filepath.Abs(filepath.Join(opt.Out, entry))
	}
	return opt.kernel(entry)
}
