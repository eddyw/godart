// Command surface measures how much of each standard library package the
// compiler surface holds. `make surface` reports the current exported names.
//
// The measure loads each package 2 times: from the real Go source with no
// build tag, and through the overlay of the compiler. It then compares the
// exported names of the package scope, the exported methods of the pointer to
// each exported named type, and the exported fields of each exported struct.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/types"
	"os"
	"os/exec"
	"runtime"
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
	absent := flag.Bool("absent", false,
		"name each standard package that the surface does not hold")
	jsonOutput := flag.Bool("json", false, "write API contracts and absent packages as JSON")
	flag.Parse()

	paths := surfacePaths()
	if *absent {
		names := absentPaths()
		fmt.Printf("%d packages of the standard library stand outside the surface.\n\n",
			len(names))
		for _, n := range names {
			fmt.Println(n)
		}
		return
	}
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
	if *jsonOutput {
		report := APIReport{GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
			Absent: absentPaths()}
		for _, path := range paths {
			report.Packages = append(report.Packages, compareAPI(path, real[path], held[path]))
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
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
	fmt.Printf("\n%d packages of %d hold each exported name, and %d hold a part.\n",
		whole, len(paths), partial)
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
	// A native package that also holds a patch stands 2 times in the surface,
	// because Surface joins the overlay list and the native list.
	seen := map[string]bool{}
	var out []string
	for _, p := range overlay.Surface() {
		if seen[p] || overlay.Intrinsic(p) || strings.HasPrefix(p, "internal/") ||
			strings.Contains(p, "/internal") {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// absentPaths gives each package of the standard library that the surface does
// not hold. An internal package and a vendored package each stand outside the
// measure, because no package of Go stands behind them.
func absentPaths() []string {
	held := map[string]bool{}
	for _, p := range overlay.Surface() {
		held[p] = true
	}
	out, err := exec.Command("go", "list", "std").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		p := strings.TrimSpace(line)
		first, _, _ := strings.Cut(p, "/")
		if p == "" || held[p] || first == "internal" || first == "vendor" ||
			strings.Contains(p, "/internal/") || strings.Contains(p, "/vendor/") ||
			strings.Contains(first, ".") {
			continue
		}
		names = append(names, p)
	}
	sort.Strings(names)
	return names
}

func loadReal(paths []string) (map[string]*types.Scope, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports |
			packages.NeedDeps,
	}, paths...)
	if err != nil {
		return nil, fmt.Errorf("the real load failed: %w", err)
	}
	var diagnostics []string
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		for _, err := range p.Errors {
			diagnostics = append(diagnostics, err.Error())
		}
	})
	if len(diagnostics) > 0 {
		return nil, fmt.Errorf("the real load failed: %s", strings.Join(diagnostics, "\n"))
	}
	return scopes(pkgs), nil
}

func loadOverlay(paths []string) (map[string]*types.Scope, error) {
	found, err := load.Load(".", paths...)
	if err != nil {
		return nil, fmt.Errorf("the overlay load failed: %w", err)
	}
	return scopes(found.Packages), nil
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
		methodType := types.Type(types.NewPointer(named))
		if _, ok := named.Underlying().(*types.Interface); ok {
			methodType = named
		}
		set := types.NewMethodSet(methodType)
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
