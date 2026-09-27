package driver_test

import (
	"encoding/json"
	"go/build"
	"os"
	"slices"
	"testing"

	"github.com/eddyw/godart/internal/parity"
)

func TestLanguageTargets(t *testing.T) {
	data, err := os.ReadFile("../parity/testdata/language-targets.json")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		t.Fatal(err)
	}
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if slices.Contains(names, c.Name) {
			cases = append(cases, c)
		}
	}
	if len(cases) != len(names) {
		t.Fatal("target manifest contains an unknown or duplicate fixture")
	}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inv, cases) })
	}
}

func TestLanguageHostTargets(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.HostTools {
			if c.Adapter != "single-file" {
				t.Fatalf("host tool recipe changed: %s: %s", c.Name, c.Recipe)
			}
			cases = append(cases, c)
		}
	}
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inv, cases) })
	}
}

func TestGeneratorCompanionTargets(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.Adapter == "generated-file" && len(c.GeneratorFiles) > 0 {
			cases = append(cases, c)
		}
	}
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inv, cases) })
	}
}

func TestRunRecipeTargets(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.Adapter == "run-recipe" {
			cases = append(cases, c)
		}
	}
	if len(cases) != 19 {
		t.Fatalf("the original run recipes changed: got %d, want 19", len(cases))
	}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			runLanguageCases(t, target, inv, cases)
		})
	}
}
