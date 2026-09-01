// Command surface measures how much of each standard library package the
// compiler surface holds. `make surface` runs it, and `docs/support.md` holds
// the numbers of the last run.
//
// The measure loads each package 2 times: from the real Go source with no
// build tag, and through the overlay of the compiler. It then compares the
// exported names of the package scope, the exported methods of the pointer to
// each exported named type, and the exported fields of each exported struct.
package main

import (
	"flag"
	"fmt"
	"go/types"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/overlay"
	"golang.org/x/tools/go/packages"
)

func main() {
	full := flag.Bool("gaps", false, "name each missing exported name")
	only := flag.String("pkg", "", "measure 1 package only")
	flag.Parse()

	paths := surfacePaths()
	if *only != "" {
		if !slices.Contains(paths, *only) {
			fmt.Fprintf(os.Stderr, "the package %s is outside the surface\n", *only)
			os.Exit(1)
		}
		paths = []string{*only}
	}

	real, err := loadReal(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	held, err := loadOverlay(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var partial, whole int
	for _, path := range paths {
		want := names(real[path])
		got := names(held[path])
		var gaps []string
		for _, n := range want {
			if !slices.Contains(got, n) {
				gaps = append(gaps, n)
			}
		}
		if len(gaps) == 0 {
			whole++
			continue
		}
		partial++
		fmt.Printf("%-24s %3d/%-3d %3d%%", path, len(want)-len(gaps), len(want),
			(len(want)-len(gaps))*100/max(len(want), 1))
		if *full {
			fmt.Printf("  %s", strings.Join(gaps, " "))
		} else {
			fmt.Printf("  %s", strings.Join(first(gaps, 6), " "))
		}
		fmt.Println()
	}
	fmt.Printf("\n%d packages hold each exported name, and %d hold a part.\n",
		whole, partial)
}

func first(list []string, n int) []string {
	if len(list) <= n {
		return list
	}
	return append(slices.Clone(list[:n]), "...")
}

// surfacePaths gives each package of the surface that the measure can load
// from the real Go source. The interop package and each internal package have
// no counterpart to compare.
func surfacePaths() []string {
	var out []string
	for _, p := range overlay.Surface() {
		if overlay.Intrinsic(p) || strings.HasPrefix(p, "internal/") ||
			strings.Contains(p, "/internal/") {
			continue
		}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func loadReal(paths []string) (map[string]*types.Scope, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports |
			packages.NeedDeps,
	}, paths...)
	if err != nil {
		return nil, fmt.Errorf("the real load failed: %w", err)
	}
	return scopes(pkgs), nil
}

func loadOverlay(paths []string) (map[string]*types.Scope, error) {
	pkgs, err := load.Load(".", paths...)
	if err != nil {
		return nil, fmt.Errorf("the overlay load failed: %w", err)
	}
	return scopes(pkgs), nil
}

func scopes(pkgs []*packages.Package) map[string]*types.Scope {
	out := map[string]*types.Scope{}
	for _, p := range pkgs {
		if p.Types == nil {
			continue
		}
		out[p.PkgPath] = p.Types.Scope()
	}
	return out
}

// names gives each exported name of a package scope, each exported method of
// the pointer to an exported named type, and each exported field of an
// exported struct.
func names(scope *types.Scope) []string {
	if scope == nil {
		return nil
	}
	var out []string
	for _, n := range scope.Names() {
		obj := scope.Lookup(n)
		if !obj.Exported() {
			continue
		}
		out = append(out, n)
		named, ok := types.Unalias(obj.Type()).(*types.Named)
		if !ok || !isTypeName(obj) {
			continue
		}
		set := types.NewMethodSet(types.NewPointer(named))
		for i := 0; i < set.Len(); i++ {
			m := set.At(i).Obj()
			if m.Exported() {
				out = append(out, n+"."+m.Name())
			}
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if f.Exported() {
				out = append(out, n+"."+f.Name())
			}
		}
	}
	sort.Strings(out)
	return slices.Compact(out)
}

func isTypeName(obj types.Object) bool {
	_, ok := obj.(*types.TypeName)
	return ok
}
