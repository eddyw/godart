package parity

import (
	"encoding/json"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func readManifest(t *testing.T, name string, dst any) {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatal(err)
	}
}

func TestCoverageManifest(t *testing.T) {
	var inventory Inventory
	readManifest(t, runtime.GOOS+"-"+runtime.GOARCH+".json", &inventory)
	var coverage []Coverage
	readManifest(t, "coverage-manifest.json", &coverage)
	var suites []struct {
		Package, Run string
		NativeTags   []string `json:"native_tags"`
	}
	readManifest(t, "vm-suites.json", &suites)
	selected := map[string]bool{}
	tags := map[string][]string{}
	for _, s := range suites {
		if selected[s.Package] {
			t.Errorf("duplicate suite: %s", s.Package)
		}
		if s.Run != "" {
			t.Errorf("unclassified suite filter: %s", s.Package)
		}
		selected[s.Package] = true
		tags[s.Package] = s.NativeTags
	}
	byPackage := map[string]Package{}
	for _, p := range inventory.Packages {
		byPackage[p.Path] = p
	}
	for _, c := range coverage {
		p, ok := byPackage[c.Package]
		if !ok {
			t.Errorf("duplicate or unknown coverage: %s", c.Package)
			continue
		}
		delete(byPackage, c.Package)
		switch c.Status {
		case "vm-suite":
			if !slices.Equal(c.NativeTags, tags[c.Package]) {
				t.Errorf("changed native build tags: %s", c.Package)
			}
			if !selected[c.Package] {
				t.Errorf("lost suite: %s", c.Package)
			}
			if len(c.NativeTags) > 0 {
				ctx := build.Default
				ctx.BuildTags = append(slices.Clone(inventory.BuildTags), c.NativeTags...)
				for _, test := range p.Tests {
					eligible, err := ctx.MatchFile(filepath.Join(ctx.GOROOT, "src", p.Path), test.File)
					if err != nil || eligible != test.Eligible {
						t.Errorf("native tags change test eligibility: %s/%s: %v", p.Path, test.File, err)
					}
				}
			}
			delete(selected, c.Package)
		case "compile-blocked":
			if c.Diagnostic == "" {
				t.Errorf("blocker lacks diagnostic: %s", c.Package)
			}
		case "no-runnable-tests":
			for _, test := range p.Tests {
				if test.Kind != "Benchmark" {
					t.Errorf("unclassified test: %s/%s", c.Package, test.Name)
				}
			}
		default:
			t.Errorf("unknown coverage status: %+v", c)
		}
	}
	for name := range byPackage {
		t.Errorf("package absent from coverage: %s", name)
	}
	for name := range selected {
		t.Errorf("suite absent from coverage: %s", name)
	}
}

func TestExceptionPolicies(t *testing.T) {
	for _, platform := range []string{"darwin-arm64", "linux-amd64", "short-darwin-arm64", "short-linux-amd64"} {
		var policies map[string][]Exception
		readManifest(t, "exceptions-"+platform+".json", &policies)
		for pkg, rules := range policies {
			seen := map[string]bool{}
			for _, rule := range rules {
				if seen[rule.Test] || rule.Test == "" || strings.TrimSpace(rule.Reason) == "" {
					t.Errorf("invalid exception: %s/%s/%s", platform, pkg, rule.Test)
				}
				seen[rule.Test] = true
			}
		}
	}
}

func TestTargetCoverage(t *testing.T) {
	var coverage []Coverage
	readManifest(t, "coverage-manifest.json", &coverage)
	var targets []struct {
		Package string
		Targets []string
		Run     string
		Exclude map[string]map[string]string
	}
	readManifest(t, "targets.json", &targets)
	var inventory Inventory
	readManifest(t, runtime.GOOS+"-"+runtime.GOARCH+".json", &inventory)
	byPackage := map[string]Coverage{}
	for _, c := range coverage {
		if len(c.Targets) > 0 {
			byPackage[c.Package] = c
		}
	}
	for _, target := range targets {
		c, ok := byPackage[target.Package]
		if !ok || !slices.Equal(c.Targets, target.Targets) || c.TargetRun != target.Run {
			t.Errorf("changed or duplicate target coverage: %s", target.Package)
		}
		delete(byPackage, target.Package)
		names := map[string]bool{}
		for _, p := range inventory.Packages {
			if p.Path == target.Package {
				for _, test := range p.Tests {
					if test.Eligible {
						names[test.Name] = true
					}
				}
			}
		}
		for platform, exclusions := range target.Exclude {
			if !slices.Contains(target.Targets, platform) {
				t.Errorf("exclusion for undeclared target: %s/%s", target.Package, platform)
			}
			for name, reason := range exclusions {
				if !names[name] || strings.TrimSpace(reason) == "" {
					t.Errorf("unclassified target exclusion: %s/%s/%s", target.Package, platform, name)
				}
			}
		}
	}
	for pkg := range byPackage {
		t.Errorf("lost target coverage: %s", pkg)
	}
}
