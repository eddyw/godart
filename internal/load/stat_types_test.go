package load_test

import (
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
	"golang.org/x/tools/go/packages"
)

func TestNativeStatTypes(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/programs/statsys")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"} {
		t.Run(target, func(t *testing.T) {
			goos, arch, _ := strings.Cut(target, "/")
			t.Setenv("GOOS", goos)
			t.Setenv("GOARCH", arch)
			t.Setenv("CGO_ENABLED", "0")
			native, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedTypes, Dir: dir}, "syscall")
			if err != nil {
				t.Fatal(err)
			}
			if len(native) != 1 || len(native[0].Errors) != 0 {
				t.Fatalf("native syscall package: %v", native)
			}
			result, err := load.Load(dir, ".")
			if err != nil {
				t.Fatal(err)
			}
			got := result.Packages[0].Imports["syscall"].Types.Scope()
			want := native[0].Types.Scope()
			for _, name := range []string{"Stat_t", "Timespec"} {
				a, b := got.Lookup(name), want.Lookup(name)
				if a == nil || b == nil {
					t.Fatalf("missing %s: Gart %v, Go %v", name, a, b)
				}
				format := func(p *types.Package) string { return p.Path() }
				actual := types.TypeString(a.Type().Underlying(), format)
				expected := types.TypeString(b.Type().Underlying(), format)
				if actual != expected {
					t.Errorf("%s differs: Gart %s, Go %s", name, actual, expected)
				}
			}
		})
	}
}
