package driver_test

import (
	"encoding/json"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	harness "github.com/eddyw/godart/internal/overlay/std/testing"
	"github.com/eddyw/godart/internal/parity"
	"golang.org/x/mod/sumdb/dirhash"
)

type moduleFixture struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Sum      string `json:"sum"`
	GoModSum string `json:"go_mod_sum"`
}

func externalFixtures(t *testing.T, output string) []moduleFixture {
	t.Helper()
	matches := regexp.MustCompile(`fetching ([A-Za-z0-9./_-]+)@([A-Za-z0-9.+_-]+)`).FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil
	}
	data, err := os.ReadFile("../parity/testdata/external-fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var pinned []moduleFixture
	if err := json.Unmarshal(data, &pinned); err != nil {
		t.Fatal(err)
	}
	var used []moduleFixture
	seen := map[string]bool{}
	for _, match := range matches {
		module := match[1] + "@" + match[2]
		if seen[module] {
			continue
		}
		seen[module] = true
		index := slices.IndexFunc(pinned, func(f moduleFixture) bool { return f.Path == match[1] && f.Version == match[2] })
		if index < 0 {
			t.Fatalf("unrecorded external fixture: %s", module)
		}
		var actual struct{ Path, Version, Sum, GoModSum, Dir string }
		text := runProgram(t, "", "go", "mod", "download", "-json", module)
		if err := json.Unmarshal([]byte(text), &actual); err != nil {
			t.Fatal(err)
		}
		want := pinned[index]
		if actual.Path != want.Path || actual.Version != want.Version || actual.Sum != want.Sum || actual.GoModSum != want.GoModSum {
			t.Fatalf("external fixture checksum changed: %s", module)
		}
		sum, err := dirhash.HashDir(actual.Dir, module, dirhash.Hash1)
		if err != nil || sum != want.Sum {
			t.Fatalf("external fixture contents changed: %s: %v", module, err)
		}
		used = append(used, want)
	}
	slices.SortFunc(used, func(a, b moduleFixture) int { return strings.Compare(a.Path, b.Path) })
	return used
}

type upstreamEvidence struct {
	ImporterInputs    *importerInputs     `json:"importer_inputs,omitempty"`
	TargetExclusions  map[string]string   `json:"target_exclusions,omitempty"`
	Selection         string              `json:"selection"`
	ResolverInputs    *resolverInputs     `json:"resolver_inputs,omitempty"`
	CertificateInputs *certificateInputs  `json:"certificate_inputs,omitempty"`
	NativeImages      []nativeImage       `json:"native_images,omitempty"`
	ExternalFixtures  []moduleFixture     `json:"external_fixtures,omitempty"`
	NativeArgs        []string            `json:"native_args"`
	NativeTrimPath    bool                `json:"native_trimpath"`
	Short             bool                `json:"short"`
	Timeout           string              `json:"timeout"`
	GoVersion         string              `json:"go_version"`
	NodeVersion       string              `json:"node_version,omitempty"`
	GartBuildTags     []string            `json:"gart_build_tags"`
	TrimPath          bool                `json:"trimpath"`
	Environment       map[string]string   `json:"environment"`
	DartVersion       string              `json:"dart_version"`
	GOOS              string              `json:"goos"`
	GOARCH            string              `json:"goarch"`
	BuildTags         []string            `json:"build_tags"`
	ToolTags          []string            `json:"tool_tags"`
	Cgo               bool                `json:"cgo"`
	SourceSHA256      string              `json:"source_sha256"`
	Fixtures          []parity.SourceFile `json:"fixtures"`
	Expected          []string            `json:"expected"`
	Skip              string              `json:"skip"`
	Target            string              `json:"target"`
	Package           string              `json:"package"`
	Filter            string              `json:"filter"`
	Native            parity.Result       `json:"native"`
	Gart              parity.Result       `json:"gart"`
	Comparison        parity.Comparison   `json:"comparison"`
}

func upstreamOmissions(t *testing.T, pkg string) map[string]parity.Omission {
	t.Helper()
	data, err := os.ReadFile("../parity/testdata/omitted-tests.json")
	if err != nil {
		t.Fatal(err)
	}
	var omitted map[string]map[string]parity.Omission
	if err := json.Unmarshal(data, &omitted); err != nil {
		t.Fatal(err)
	}
	return omitted[pkg]
}

func upstreamSkip(t *testing.T, pkg string) string {
	var patterns []string
	for name := range upstreamOmissions(t, pkg) {
		parts := strings.Split(name, "/")
		for i := range parts {
			parts[i] = "^" + regexp.QuoteMeta(parts[i]) + "$"
		}
		patterns = append(patterns, strings.Join(parts, "/"))
	}
	slices.Sort(patterns)
	return strings.Join(patterns, "|")
}

func upstreamLimit() time.Duration {
	if testing.Short() {
		return runLimit
	}
	return 75 * time.Minute
}

func upstreamExtraArgs(pkg string) []string {
	if pkg == "debug/buildinfo" && !testing.Short() {
		// The fixture parses native SDK output. Its run filter selects the SDK platform.
		return []string{"-all"}
	}
	return nil
}

func compareUpstream(t *testing.T, name, pkg, filter, target, dir, dartText string, dartExit int, images ...nativeImage) {
	t.Helper()
	compareUpstreamInputs(t, name, pkg, filter, target, dir, dartText, dartExit, nil, nil, images...)
}

func compareUpstreamInputs(t *testing.T, name, pkg, filter, target, dir, dartText string, dartExit int, inputs *certificateInputs, resolver *resolverInputs, images ...nativeImage) {
	t.Helper()
	compareUpstreamSDKInputs(t, name, pkg, filter, target, dir, dartText, dartExit, inputs, resolver, nil, images...)
}

func compareUpstreamSDKInputs(t *testing.T, name, pkg, filter, target, dir, dartText string, dartExit int, inputs *certificateInputs, resolver *resolverInputs, importer *importerInputs, images ...nativeImage) {
	t.Helper()
	var inventory parity.Inventory
	nameOfInventory, err := parity.InventoryName(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join("..", "parity", "testdata", nameOfInventory))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(manifest, &inventory); err != nil {
		t.Fatal(err)
	}
	compareUpstreamSDKInventory(t, name, pkg, filter, target, dir, dartText, dartExit, inventory, inputs, resolver, nil, importer, images...)
}

func compareUpstreamInventory(t *testing.T, name, pkg, filter, target, dir, dartText string, dartExit int, inventory parity.Inventory, inputs *certificateInputs, resolver *resolverInputs, targetExclusions map[string]string, images ...nativeImage) {
	t.Helper()
	compareUpstreamSDKInventory(t, name, pkg, filter, target, dir, dartText, dartExit, inventory, inputs, resolver, targetExclusions, nil, images...)
}

func compareUpstreamSDKInventory(t *testing.T, name, pkg, filter, target, dir, dartText string, dartExit int, inventory parity.Inventory, inputs *certificateInputs, resolver *resolverInputs, targetExclusions map[string]string, importer *importerInputs, images ...nativeImage) {
	t.Helper()
	nativeTags := slices.Clone(build.Default.BuildTags)
	args := []string{"test", "-json", "-count=1", "-timeout=" + upstreamLimit().String()}
	if inputs != nil {
		args = append(args, "-overlay="+inputs.Overlay)
	}
	var suites []struct {
		Package    string   `json:"package"`
		NativeTags []string `json:"native_tags"`
	}
	suiteData, err := os.ReadFile("../parity/testdata/vm-suites.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(suiteData, &suites); err != nil {
		t.Fatal(err)
	}
	for _, suite := range suites {
		if suite.Package == pkg && len(suite.NativeTags) > 0 {
			nativeTags = append(nativeTags, suite.NativeTags...)
			args = append(args, "-tags="+strings.Join(nativeTags, ","))
		}
	}
	if testing.Short() {
		args = append(args, "-short")
	}
	if filter != "" {
		args = append(args, "-run", filter)
	}
	args = append(args, ".")
	if extra := upstreamExtraArgs(pkg); len(extra) != 0 {
		args = append(args, "-args")
		args = append(args, extra...)
	}
	nativeText, err, ok := output(dir, nil, upstreamLimit(), "go", args...)
	nativeExit := 0
	if !ok {
		nativeExit = 124
	}
	if err != nil && ok {
		if e, ok := err.(*exec.ExitError); ok {
			nativeExit = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	base := ""
	if evidence := os.Getenv("GART_PARITY_REPORT_DIR"); evidence != "" {
		if err := os.MkdirAll(evidence, 0755); err != nil {
			t.Fatal(err)
		}
		base = filepath.Join(evidence, strings.ReplaceAll(name, "/", "_"))
		if target != "vm" {
			base += "." + target
		}
		for suffix, text := range map[string][]byte{".native.jsonl": nativeText, ".gart.log": []byte(dartText)} {
			if err := os.WriteFile(base+suffix, text, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	native, err := parity.NativeResults(strings.NewReader(string(nativeText)), nativeExit)
	if err != nil {
		t.Fatal(err)
	}
	gart, err := parity.GartResults(strings.NewReader(dartText), dartExit)
	if err != nil {
		t.Fatal(err)
	}
	native, err = parity.CanonicalNames(pkg, native)
	if err != nil {
		t.Fatal(err)
	}
	gart, err = parity.CanonicalNames(pkg, gart)
	if err != nil {
		t.Fatal(err)
	}
	var policies map[string][]parity.Exception
	prefix := "exceptions-"
	if testing.Short() {
		prefix += "short-"
	}
	path := filepath.Join("..", "parity", "testdata", prefix+runtime.GOOS+"-"+runtime.GOARCH+".json")
	data, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(data, &policies); err != nil {
			t.Fatal(err)
		}
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var expected []string
	found := false
	for _, p := range inventory.Packages {
		if p.Path != pkg {
			continue
		}
		found = true
		if importer != nil {
			file := filepath.Base(importer.Source.Path)
			index := slices.IndexFunc(p.Files, func(f parity.SourceFile) bool { return f.Path == file })
			if index < 0 || p.Files[index].SHA256 != importer.Source.SHA256 {
				t.Fatal("importer inputs differ from the SDK inventory")
			}
		}
		if resolver != nil {
			index := slices.IndexFunc(p.Files, func(f parity.SourceFile) bool { return f.Path == "main_test.go" })
			if index < 0 || p.Files[index].SHA256 != resolver.SourceSHA256 {
				t.Fatal("resolver setup differs from the SDK inventory")
			}
		}
		for _, test := range p.Tests {
			if test.Eligible && test.Kind != "Benchmark" && harness.MatchName(test.Name, filter, regexp.MatchString) {
				expected = append(expected, test.Name)
			}
		}
	}
	if !found {
		t.Fatalf("package missing from inventory: %s", pkg)
	}
	var selectedRules []parity.Exception
	rules := policies[pkg]
	if name != pkg {
		rules = append(slices.Clone(rules), policies[name]...)
	}
	for _, rule := range rules {
		if harness.MatchName(rule.Test, filter, regexp.MatchString) {
			selectedRules = append(selectedRules, rule)
		}
	}
	for root, omission := range upstreamOmissions(t, pkg) {
		if !harness.MatchName(root, filter, regexp.MatchString) {
			continue
		}
		found := false
		for name, status := range native.Outcomes {
			if name == root || strings.HasPrefix(name, root+"/") {
				found = true
				selectedRules = append(selectedRules, parity.Exception{Test: name, Native: status, Gart: "", Reason: omission.Reason})
			}
		}
		if !found {
			t.Errorf("unused upstream exclusion: %s/%s", pkg, root)
		}
	}
	external := externalFixtures(t, string(nativeText)+"\n"+dartText)
	comparison := parity.Compare(native, gart, expected, selectedRules)
	if base != "" {
		dartVersion := strings.TrimSpace(runProgram(t, "", "dart", "--version"))
		nodeVersion := ""
		if target == "js" || target == "wasm" {
			nodeVersion = strings.TrimSpace(runProgram(t, "", "node", "--version"))
		}
		env := map[string]string{"GODEBUG": os.Getenv("GODEBUG"), "GOEXPERIMENT": os.Getenv("GOEXPERIMENT")}
		if pkg == "go/build" || pkg == "debug/buildinfo" || pkg == "internal/testenv" || importer != nil || pkg == "go/importer" || pkg == "go/internal/gccgoimporter" || pkg == "internal/types/errors" {
			env["GOROOT"] = build.Default.GOROOT
		}
		data, err := json.MarshalIndent(upstreamEvidence{ImporterInputs: importer, TargetExclusions: targetExclusions, Selection: name, ResolverInputs: resolver, CertificateInputs: inputs, NativeImages: images, ExternalFixtures: external, NativeArgs: args, NativeTrimPath: false, Short: testing.Short(), Timeout: upstreamLimit().String(), NodeVersion: nodeVersion, GartBuildTags: []string{"dart"}, TrimPath: true, Environment: env, GoVersion: runtime.Version(), DartVersion: dartVersion, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, BuildTags: nativeTags, ToolTags: build.Default.ToolTags, Cgo: build.Default.CgoEnabled, SourceSHA256: inventory.SourceSHA256, Fixtures: inventory.Fixtures, Expected: expected, Skip: upstreamSkip(t, pkg), Target: target, Package: pkg, Filter: filter, Native: native, Gart: gart, Comparison: comparison}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(base+".json", data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("upstream parity: %d passes, %d partial, %d explicit exclusions, %d target exclusions", len(comparison.Passed), len(comparison.Partial), len(comparison.Excluded), len(targetExclusions))
	if len(comparison.Problems) > 0 {
		t.Error(fmt.Sprintf("upstream parity failed:\n%s", strings.Join(comparison.Problems, "\n")))
	}
}
