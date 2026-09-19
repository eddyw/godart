package sdk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestRepositoryVersions(t *testing.T) {
	versions, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	module, err := modfile.Parse("go.mod", read("go.mod"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if module.Go.Version != versions.Go.Baseline {
		t.Fatalf("go.mod uses %s, manifest baseline is %s", module.Go.Version, versions.Go.Baseline)
	}
	var fvm struct {
		Flutter string `json:"flutter"`
	}
	if err := json.Unmarshal(read(".fvmrc"), &fvm); err != nil {
		t.Fatal(err)
	}
	if fvm.Flutter != versions.Flutter.Development {
		t.Fatalf("FVM uses %s, manifest selects %s", fvm.Flutter, versions.Flutter.Development)
	}
	for _, version := range []string{versions.Dart.Baseline, versions.Flutter.Dart} {
		if !slices.Contains(versions.Dart.Resident, version) {
			t.Errorf("Dart %s is absent from the resident versions", version)
		}
	}
	docker := string(read("scripts/Dockerfile.ci"))
	for _, image := range []string{"golang:" + versions.Go.Baseline, "dart:" + versions.Dart.Baseline} {
		if !strings.Contains(docker, "FROM "+image+"@sha256:") {
			t.Errorf("CI image does not pin %s", image)
		}
	}
}
