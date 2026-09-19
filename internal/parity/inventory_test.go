package parity

import (
	"encoding/json"
	"go/build"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpstreamInventory(t *testing.T) {
	name, err := InventoryName(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", name))
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
	module := filepath.Join(root, "src", "go.mod")
	if err := os.WriteFile(module, []byte("module std\n"), 0644); err != nil {
		t.Fatal(err)
	}
	withModule, err := Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := filepath.Join(root, "src", "_go.mod")
	if err := os.WriteFile(duplicate, []byte("module std\n"), 0644); err != nil {
		t.Fatal(err)
	}
	withDuplicate, err := Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if withModule.SourceSHA256 != withDuplicate.SourceSHA256 {
		t.Fatal("identical toolchain module duplicate changed identity")
	}
	if err := os.WriteFile(duplicate, []byte("module changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	withChange, err := Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if withModule.SourceSHA256 == withChange.SourceSHA256 {
		t.Fatal("different toolchain module duplicate escaped identity")
	}
	before := withChange.SourceSHA256
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
