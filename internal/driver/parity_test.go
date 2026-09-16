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
)

type upstreamEvidence struct {
	NativeArgs     []string            `json:"native_args"`
	NativeTrimPath bool                `json:"native_trimpath"`
	Short          bool                `json:"short"`
	Timeout        string              `json:"timeout"`
	GoVersion      string              `json:"go_version"`
	NodeVersion    string              `json:"node_version,omitempty"`
	GartBuildTags  []string            `json:"gart_build_tags"`
	TrimPath       bool                `json:"trimpath"`
	Environment    map[string]string   `json:"environment"`
	DartVersion    string              `json:"dart_version"`
	GOOS           string              `json:"goos"`
	GOARCH         string              `json:"goarch"`
	BuildTags      []string            `json:"build_tags"`
	ToolTags       []string            `json:"tool_tags"`
	Cgo            bool                `json:"cgo"`
	SourceSHA256   string              `json:"source_sha256"`
	Fixtures       []parity.SourceFile `json:"fixtures"`
	Expected       []string            `json:"expected"`
	Skip           string              `json:"skip"`
	Target         string              `json:"target"`
	Package        string              `json:"package"`
	Filter         string              `json:"filter"`
	Native         parity.Result       `json:"native"`
	Gart           parity.Result       `json:"gart"`
	Comparison     parity.Comparison   `json:"comparison"`
}

func upstreamOmissions(t *testing.T, pkg string) map[string]string {
	t.Helper()
	data, err := os.ReadFile("../parity/testdata/omitted-tests.json")
	if err != nil {
		t.Fatal(err)
	}
	var omitted map[string]map[string]string
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
	return 30 * time.Minute
}

func compareUpstream(t *testing.T, name, pkg, filter, target, dir, dartText string, dartExit int) {
	t.Helper()
	args := []string{"test", "-json", "-count=1", "-timeout=" + upstreamLimit().String()}
	if testing.Short() {
		args = append(args, "-short")
	}
	if filter != "" {
		args = append(args, "-run", filter)
	}
	args = append(args, ".")
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
	var inventory parity.Inventory
	manifest, err := os.ReadFile(filepath.Join("..", "parity", "testdata", runtime.GOOS+"-"+runtime.GOARCH+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(manifest, &inventory); err != nil {
		t.Fatal(err)
	}
	var expected []string
	found := false
	for _, p := range inventory.Packages {
		if p.Path != pkg {
			continue
		}
		found = true
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
	for _, rule := range policies[name] {
		if harness.MatchName(rule.Test, filter, regexp.MatchString) {
			selectedRules = append(selectedRules, rule)
		}
	}
	for root, reason := range upstreamOmissions(t, pkg) {
		if !harness.MatchName(root, filter, regexp.MatchString) {
			continue
		}
		found := false
		for name, status := range native.Outcomes {
			if name == root || strings.HasPrefix(name, root+"/") {
				found = true
				selectedRules = append(selectedRules, parity.Exception{Test: name, Native: status, Gart: "", Reason: reason})
			}
		}
		if !found {
			t.Errorf("unused upstream exclusion: %s/%s", pkg, root)
		}
	}
	comparison := parity.Compare(native, gart, expected, selectedRules)
	if base != "" {
		dartVersion := strings.TrimSpace(runProgram(t, "", "dart", "--version"))
		nodeVersion := ""
		if target == "js" || target == "wasm" {
			nodeVersion = strings.TrimSpace(runProgram(t, "", "node", "--version"))
		}
		data, err := json.MarshalIndent(upstreamEvidence{NativeArgs: args, NativeTrimPath: false, Short: testing.Short(), Timeout: upstreamLimit().String(), NodeVersion: nodeVersion, GartBuildTags: []string{"dart"}, TrimPath: true, Environment: map[string]string{"GODEBUG": os.Getenv("GODEBUG"), "GOEXPERIMENT": os.Getenv("GOEXPERIMENT")}, GoVersion: runtime.Version(), DartVersion: dartVersion, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, BuildTags: build.Default.BuildTags, ToolTags: build.Default.ToolTags, Cgo: build.Default.CgoEnabled, SourceSHA256: inventory.SourceSHA256, Fixtures: inventory.Fixtures, Expected: expected, Skip: upstreamSkip(t, pkg), Target: target, Package: pkg, Filter: filter, Native: native, Gart: gart, Comparison: comparison}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(base+".json", data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("upstream parity: %d passes, %d explicit exclusions", len(comparison.Passed), len(comparison.Excluded))
	if len(comparison.Problems) > 0 {
		t.Error(fmt.Sprintf("upstream parity failed:\n%s", strings.Join(comparison.Problems, "\n")))
	}
}
