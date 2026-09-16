package parity

import (
	"encoding/json"
	"go/build"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestUpstreamInventory(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", runtime.GOOS+"-"+runtime.GOARCH+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected Inventory
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	actual, err := Discover(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&expected, actual) {
		t.Fatal("upstream source, test inventory, platform configuration, or test patches changed without review")
	}
}

func TestInventoryPreservesIneligibleTests(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src", "path")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib/time"), 0755); err != nil {
		t.Fatal(err)
	}
	for path, text := range map[string]string{
		filepath.Join(root, "lib/time/zoneinfo.zip"): "fixture",
		filepath.Join(root, "VERSION"):               "go1.27.0\n",
		filepath.Join(dir, "a_test.go"):              "package path\nimport \"testing\"\nfunc TestValue(t *testing.T) {}\nfunc FuzzValue(f *testing.F) {}\nfunc ExampleValue() {\n// Output: ok\n}\n",
		filepath.Join(dir, "excluded_test.go"):       "//go:build never_selected\n\npackage path\nimport \"testing\"\nfunc TestExcluded(t *testing.T) {}\n",
	} {
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := build.Default
	ctx.GOROOT = root
	inv, err := Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Packages) != 1 || len(inv.Packages[0].Tests) != 4 {
		t.Fatalf("%+v", inv)
	}
	for _, test := range inv.Packages[0].Tests {
		if test.Name == "TestExcluded" && test.Eligible {
			t.Fatal("ineligible test accepted")
		}
	}
	before := inv.SourceSHA256
	if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package path\n"), 0644); err != nil {
		t.Fatal(err)
	}
	inv, err = Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if inv.SourceSHA256 == before {
		t.Fatal("source drift escaped detection")
	}
}
