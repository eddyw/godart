package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func fieldtrackFixtures(t *testing.T, adapter string, count int) (*parity.LanguageInventory, []parity.LanguageCase) {
	t.Helper()
	t.Setenv("GOEXPERIMENT", "fieldtrack")
	inventory, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, fixture := range inventory.Cases {
		if fixture.Adapter == adapter {
			cases = append(cases, fixture)
		}
	}
	if len(cases) != count {
		t.Fatalf("original %s fixtures changed: got %d, want %d", adapter, len(cases), count)
	}
	return inventory, cases
}

func TestNoInterfaceUpstream(t *testing.T) {
	inventory, cases := fieldtrackFixtures(t, "fieldtrack-file", 2)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inventory, cases) })
	}
}

func TestNoInterfaceUpstreamRejections(t *testing.T) {
	inventory, cases := fieldtrackFixtures(t, "fieldtrack-rejection", 2)
	runDiagnosticCases(t, inventory, cases)
}

func TestNoInterfaceAssignments(t *testing.T) {
	t.Setenv("GOEXPERIMENT", "fieldtrack")
	cases := map[string]string{
		"variable":                 `var _ I = T{}`,
		"assignment":               `func f(){var x I; x=T{}; _=x}`,
		"short assignment":         `func f(){var x I; x,y:=T{},1; _,_=x,y}`,
		"conversion":               `var _ = I(T{})`,
		"argument":                 `func take(I){}; func f(){take(T{})}`,
		"variadic":                 `func take(...I){}; func f(){take(T{})}`,
		"tuple argument":           `func take(I,I){}; func f(){take(pair())}`,
		"tuple variable":           `var x,y I = pair()`,
		"tuple assignment":         `func f(){var x,y I; x,y=pair(); _,_=x,y}`,
		"return":                   `func f() I {return T{}}`,
		"tuple return":             `func f()(I,I){return pair()}`,
		"function literal":         `var _=func() I {return T{}}`,
		"array":                    `var _=[1]I{T{}}`,
		"slice":                    `var _=[]I{0:T{}}`,
		"map key":                  `var _=map[I]int{T{}:0}`,
		"map value":                `var _=map[int]I{0:T{}}`,
		"struct":                   `var _=struct{F I}{T{}}`,
		"implicit pointer literal": `var _=[]*struct{F I}{{F:T{}}}`,
		"send":                     `func f(ch chan I){ch<-T{}}`,
		"comparison":               `func f(x I)bool{return x==T{}}`,
		"switch":                   `func f(x I){switch x {case T{}:}}`,
		"assertion":                `func f(x I){_=x.(T)}`,
		"type switch":              `func f(x I){switch x.(type){case T:}}`,
		"append":                   `func f(s []I){_=append(s,T{})}`,
		"slice range":              `func f(s []T){var v I; for _,v=range s{}; _=v}`,
		"map range":                `func f(s map[T]int){var v I; for v=range s{}; _=v}`,
		"channel range":            `func f(ch chan T){var v I; for v=range ch{}; _=v}`,
		"iterator range":           `func f(seq func(func(T)bool)){var v I; for v=range seq{}; _=v}`,
		"separated directive": `type N struct{}
//go:nointerface

func(N) M(){}
var _ I=N{}`,
		"integer range": `type N int
//go:nointerface
func(N) M(){}
func f(n N){var v I; for v=range n{}; _=v}`,
		"named iterator range": `type Yield func(T)bool
func f(seq func(Yield)){var v I; for v=range seq{}; _=v}`,

		"generic type":       `type G[X I] struct{}; var _ G[T]`,
		"generic function":   `func f[X I](X){}; func g(){f(T{})}`,
		"generic alias":      `type G[X I]=[]X; var _ G[T]`,
		"signature instance": `type G[X I] struct{}; func f(G[T]){}`,
	}
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			dir, out := t.TempDir(), t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nointerfaceassign\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "main.go"), `package main
 type I interface{M()}
 type T struct{}
 //go:nointerface
 func(T) M(){}
 func pair()(T,T){return T{},T{}}
 func main(){}
 `+cases[name])
			native := programOutcome(t, dir, "go", "build", ".")
			if native.Exit == 0 || !strings.Contains(native.Text, "nointerface") {
				t.Fatalf("unexpected native outcome: %+v", native)
			}
			err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)})
			if err == nil || !strings.Contains(err.Error(), "nointerface") || !strings.Contains(err.Error(), "main.go:") {
				t.Fatalf("expected positioned nointerface rejection, got %v", err)
			}
		})
	}
}

func TestNoInterfaceMetadata(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nointerfacemetadata\n\ngo 1.27\n")
	dependency := `package dep
 type Base struct{}
 //go:nointerface
 func(Base) Hidden() int{return 7}
 func(Base) Visible() int{return 11}
 type Generic[T any] struct{Base}
 //go:nointerface
 func(Generic[T]) GenericHidden() int{return 13}
 func(Generic[T]) GenericVisible() int{return 17}
 `
	if err := os.Mkdir(filepath.Join(dir, "dep"), 0755); err != nil {
		t.Fatal(err)
	}
	dependencyPath := filepath.Join(dir, "dep", "dep.go")
	writeFile(t, dependencyPath, dependency)
	writeFile(t, filepath.Join(dir, "main.go"), `package main
 import("fmt";"reflect";"example.com/nointerfacemetadata/dep")
 type Outer struct{dep.Generic[int]}
 func visible[T interface{Visible()int}](v T)int{return v.Visible()}
 func main(){
  value:=Outer{}
  method:=value.Hidden
  generic:=value.GenericHidden
  fmt.Println("direct",method(),generic(),Outer.Hidden(value),(*Outer).Hidden(&value),dep.Generic[int].GenericHidden(dep.Generic[int]{}),visible(value))
  for _,v:=range []any{dep.Base{},dep.Generic[int]{},value,&value}{
   typ:=reflect.TypeOf(v)
   _,hidden:=v.(interface{Hidden()int})
   _,genericHidden:=v.(interface{GenericHidden()int})
   fmt.Println(typ.String(),typ.NumMethod(),hidden,genericHidden,typ.Implements(reflect.TypeOf((*interface{Hidden()int})(nil)).Elem()))
   for i:=0;i<typ.NumMethod();i++ {fmt.Println("method",typ.Method(i).Name)}
   fmt.Println("hidden value",reflect.ValueOf(v).MethodByName("Hidden").IsValid())
   fmt.Println("visible call",reflect.ValueOf(v).MethodByName("Visible").Call(nil)[0].Int())
  }
 }
 `)
	phases := []struct {
		name, experiment string
		remove           bool
		allTargets       bool
	}{
		{"default", "", false, true},
		{"enabled", "fieldtrack", false, true},
		{"disabled", "fieldtrack,nofieldtrack", false, false},
		{"enabled again", "fieldtrack", false, false},
		{"dependency directive removed", "fieldtrack", true, false},
	}
	for _, phase := range phases {
		t.Run(phase.name, func(t *testing.T) {
			t.Setenv("GOEXPERIMENT", phase.experiment)
			if phase.remove {
				writeFile(t, dependencyPath, strings.Replace(dependency, "//go:nointerface\n", "", 1))
			}
			want := programOutcome(t, dir, "go", "run", ".")
			if want.Exit != 0 {
				t.Fatalf("native failed: %+v", want)
			}
			options := driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}
			stats, err := driver.BuildWithStats(options)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(stats.Compiled, "example.com/nointerfacemetadata") {
				t.Fatalf("root did not rebuild: %+v", stats)
			}
			if phase.name == "enabled" {
				cached, err := driver.BuildWithStats(options)
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Contains(cached.Cached, "example.com/nointerfacemetadata") {
					t.Fatalf("unchanged root did not use cache: %+v", cached)
				}
			}
			prepareDart(t, out)
			targets := []string{"vm"}
			if phase.allTargets {
				targets = append(targets, "aot", "js", "wasm")
			}
			for _, target := range targets {
				t.Run(target, func(t *testing.T) {
					var got programResult
					if target == "vm" {
						got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
					}
					if got != want {
						t.Fatalf("native: %+v\nGart: %+v", want, got)
					}
				})
			}
		})
	}
}

func TestNoInterfaceImportedUpstream(t *testing.T) {
	inventory, cases := fieldtrackFixtures(t, "fieldtrack-module", 1)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inventory, cases) })
	}
}

func TestNoInterfaceImportedRejection(t *testing.T) {
	t.Setenv("GOEXPERIMENT", "fieldtrack")
	dir, out := t.TempDir(), t.TempDir()
	source := filepath.Join(runtime.GOROOT(), "test", "fixedbugs", "issue30862.dir")
	if err := os.CopyFS(dir, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module issue30862.dir\n\ngo 1.27\n")
	options := driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}

	t.Run("imported rejection", func(t *testing.T) {
		writeFile(t, filepath.Join(dir, "reject.go"), `package main
  import "issue30862.dir/a"
  var _ interface{NoInterfaceMethod()} = new(a.NoitfStruct)
  `)
		native := programOutcome(t, dir, "go", "build", ".")
		if native.Exit == 0 || !strings.Contains(native.Text, "nointerface") {
			t.Fatalf("unexpected native outcome: %+v", native)
		}
		err := driver.Build(options)
		if err == nil || !strings.Contains(err.Error(), "nointerface") || !strings.Contains(err.Error(), "reject.go:") {
			t.Fatalf("expected imported nointerface rejection, got %v", err)
		}
	})
}

func TestNoInterfaceDirectivePlacement(t *testing.T) {
	for _, experiment := range []string{"fieldtrack", "nofieldtrack"} {
		t.Run(experiment, func(t *testing.T) {
			t.Setenv("GOEXPERIMENT", experiment)
			cases := map[string]string{
				"ordinary function": "//go:nointerface\n\nfunc f(){}\n",
				"variable":          "//go:nointerface\nvar x int\n",
				"type":              "//go:nointerface\ntype T struct{}\n",
				"body":              "func f(){\n//go:nointerface\n}\n",
				"end of file":       "//go:nointerface\n",
			}
			names := make([]string, 0, len(cases))
			for name := range cases {
				names = append(names, name)
			}
			slices.Sort(names)
			for _, name := range names {
				t.Run(name, func(t *testing.T) {
					dir, out := t.TempDir(), t.TempDir()
					writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nointerfaceplacement\n\ngo 1.27\n")
					writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n"+cases[name])
					native := programOutcome(t, dir, "go", "build", ".")
					err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)})
					if native.Exit == 0 {
						if err != nil {
							t.Fatal(err)
						}
						return
					}
					if !strings.Contains(native.Text, "misplaced compiler directive") {
						t.Fatalf("unexpected native outcome: %+v", native)
					}
					if err == nil || !strings.Contains(err.Error(), "misplaced compiler directive") {
						t.Fatalf("expected native directive rejection, got %v", err)
					}
				})
			}
		})
	}
}
