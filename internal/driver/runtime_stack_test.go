package driver_test

import (
	"go/build"
	"testing"

	"github.com/eddyw/godart/internal/parity"
)

func TestStackLanguage(t *testing.T) {
	inventory, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, fixture := range inventory.Cases {
		if fixture.Name == "stack" || fixture.Name == "stackobj2" {
			cases = append(cases, fixture)
		}
	}
	if len(cases) != 2 {
		t.Fatal("missing upstream stack fixtures")
	}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inventory, cases) })
	}
}
