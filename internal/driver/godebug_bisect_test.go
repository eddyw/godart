package driver_test

import (
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGODEBUGBisectState(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/bisectstate\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "math/rand"; "os")
//go:noinline
func seedA() { rand.Seed(1) }
//go:noinline
func seedB() { rand.Seed(2) }
func main() {
 if len(os.Args) > 1 && os.Args[1] == "values" {
  for _, value := range []string{"0#qy", "0#qn", "0#q!y", "0#q!n", "0#invalid", "0", "1#qy"} {
   os.Setenv("GODEBUG", "randautoseed=0,randseednop=" + value)
   rand.Seed(1)
   fmt.Println(rand.Int63())
  }
  return
 }
 calls := []func(){seedA, seedB}
 if len(os.Args) > 1 { calls[0], calls[1] = calls[1], calls[0] }
 for phase := 0; phase < 5; phase++ {
  switch phase {
  case 2: os.Unsetenv("GODEBUG")
  case 3: os.Setenv("GODEBUG", "randseednop=1")
  case 4: os.Clearenv()
  }
  os.Setenv("GODEBUG", "randseednop=0#vy")
  for repeat := 0; repeat < 2; repeat++ {
   for _, call := range calls { call() }
  }
 }
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	wantValues := programOutcome(t, dir, native, "values")
	if wantValues.Exit != 0 {
		t.Fatal(wantValues.Text)
	}
	prepareDart(t, out)
	aot := filepath.Join(out, "app")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", aot)
	marker := regexp.MustCompile(`\[bisect-match (0x[0-9a-f]+)\].*main\.(seed[AB])`)
	for _, target := range []struct {
		name, command string
		args          []string
	}{
		{"native", native, nil}, {"vm", "dart", []string{"bin/main.dart"}}, {"aot", aot, nil},
	} {
		t.Run(target.name, func(t *testing.T) {
			gotValues := programOutcome(t, out, target.command, append(slices.Clone(target.args), "values")...)
			if gotValues != wantValues {
				t.Fatalf("pattern values: got %+v, want %+v", gotValues, wantValues)
			}
			var previous map[string][]string
			for order := 0; order < 2; order++ {
				args := slices.Clone(target.args)
				if order != 0 {
					args = append(args, "reverse")
				}
				result := programOutcome(t, out, target.command, args...)
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
				found := map[string][]string{}
				for _, match := range marker.FindAllStringSubmatch(result.Text, -1) {
					found[match[2]] = append(found[match[2]], match[1])
				}
				for _, name := range []string{"seedA", "seedB"} {
					ids := found[name]
					if len(ids) != 5 {
						t.Errorf("order %d %s: got %d reports, want 5\n%s", order, name, len(ids), result.Text)
						continue
					}
					for _, id := range ids {
						if id != ids[0] {
							t.Errorf("%s ID changed after environment update: %v", name, ids)
						}
					}
					if order != 0 && !slices.Equal(previous[name], ids) {
						t.Errorf("%s IDs depend on call order: %v / %v", name, previous[name], ids)
					}
				}
				previous = found
			}
		})
	}
}
