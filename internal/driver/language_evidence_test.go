package driver_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/parity"
)

type languageExecution struct {
	Command      []string `json:"command,omitempty"`
	Ran          bool     `json:"ran"`
	Exit         int      `json:"exit"`
	TimedOut     bool     `json:"timed_out"`
	OutputSHA256 string   `json:"output_sha256,omitempty"`
	Seconds      float64  `json:"seconds"`
}

type languageOutcome struct {
	NativeDiagnostics string              `json:"native_diagnostics,omitempty"`
	GartDiagnostics   string              `json:"gart_diagnostics,omitempty"`
	NativeMatch       languageExecution   `json:"native_match"`
	GartMatch         languageExecution   `json:"gart_match"`
	NativeMismatch    string              `json:"native_mismatch,omitempty"`
	GartMismatch      string              `json:"gart_mismatch,omitempty"`
	Contract          string              `json:"contract,omitempty"`
	WrapperSHA256     string              `json:"wrapper_sha256,omitempty"`
	BuildError        string              `json:"build_error,omitempty"`
	Fixture           parity.LanguageCase `json:"fixture"`
	Status            string              `json:"status"`
	Native            languageExecution   `json:"native"`
	Gart              languageExecution   `json:"gart"`
	ExpectedSHA256    string              `json:"expected_sha256,omitempty"`
}

type languageEvidence struct {
	Contract     string             `json:"contract"`
	NodeVersion  string             `json:"node_version,omitempty"`
	Kind         string             `json:"kind"`
	GoVersion    string             `json:"go_version"`
	DartVersion  string             `json:"dart_version"`
	GOOS         string             `json:"goos"`
	GOARCH       string             `json:"goarch"`
	Target       string             `json:"target"`
	SourceSHA256 string             `json:"source_sha256"`
	ToolTags     []string           `json:"tool_tags"`
	Cgo          bool               `json:"cgo"`
	Environment  map[string]string  `json:"environment"`
	Shard        int                `json:"shard"`
	Shards       int                `json:"shards"`
	Complete     bool               `json:"complete"`
	Outcomes     []*languageOutcome `json:"outcomes"`
}

func languageDigest(text []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(text))
}

func retainLanguageEvidence(t *testing.T, target string, inv *parity.LanguageInventory, outcomes []*languageOutcome) {
	t.Helper()
	directory := os.Getenv("GART_PARITY_REPORT_DIR")
	if directory == "" {
		return
	}
	path := filepath.Join(directory, "language", fmt.Sprintf("%s-%d-%s.json", target, *shardIndex, strings.ReplaceAll(t.Name(), "/", "-")))
	report := languageEvidence{Kind: "language", GoVersion: inv.GoVersion, DartVersion: strings.TrimSpace(runProgram(t, "", "dart", "--version")), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Target: target, SourceSHA256: inv.SourceSHA256, ToolTags: inv.ToolTags, Cgo: inv.Cgo, Environment: map[string]string{"GODEBUG": os.Getenv("GODEBUG"), "GOEXPERIMENT": os.Getenv("GOEXPERIMENT"), "GO_GCFLAGS": inv.GCFlags}, Shard: *shardIndex, Shards: *shardCount, Outcomes: outcomes}
	report.Contract = "run"
	if len(outcomes) > 0 && outcomes[0].Contract != "" {
		report.Contract = outcomes[0].Contract
	}
	if target == "js" || target == "wasm" {
		report.NodeVersion = strings.TrimSpace(runProgram(t, "", "node", "--version"))
	}
	write := func() {
		t.Helper()
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write()
	t.Cleanup(func() { report.Complete = true; write() })
}
