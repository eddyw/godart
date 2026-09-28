package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/parity"
)

func TestUpstreamTimeoutReport(t *testing.T) {
	if mode := os.Getenv("GART_TIMEOUT_REPORT_CHILD"); mode != "" {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/report\n\ngo 1.27\n")
		writeFile(t, filepath.Join(dir, "report_test.go"), "package report\nimport \"testing\"\nfunc TestPass(t *testing.T) {}\nfunc TestWait(t *testing.T) {}\n")
		script := "printf '%s\\n' '--- PASS: TestPass (0.00s)'\n"
		switch mode {
		case "pass":
			script += "printf '%s\\n' '--- PASS: TestWait (0.00s)'"
		case "fail":
			script += "printf '%s\\n' '--- FAIL: TestWait (0.00s)'\nexit 1"
		case "timeout":
			script += "printf '%s\\n' '=== RUN   TestWait'\nexec sleep 30"
		}
		result := upstreamOutcome(t, dir, nil, time.Second, "sh", "-c", script)
		inventory := parity.Inventory{Packages: []parity.Package{{Path: "example.com/report", Tests: []parity.Test{
			{Name: "TestPass", Kind: "Test", Eligible: true},
			{Name: "TestWait", Kind: "Test", Eligible: true},
		}}}}
		compareUpstreamInventory(t, "timeout-report", "example.com/report", "", "vm", dir, result.Text, result.Exit, inventory, nil, nil, nil, false)
		return
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"pass", "fail", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GART_TIMEOUT_REPORT_CHILD", mode)
			t.Setenv("GART_PARITY_REPORT_DIR", dir)
			result := programOutcome(t, "", self, "-test.run=^TestUpstreamTimeoutReport$", "-test.v")
			if (result.Exit == 0) != (mode == "pass") {
				t.Fatalf("unexpected comparison exit: %+v", result)
			}
			data, err := os.ReadFile(filepath.Join(dir, "timeout-report.json"))
			if err != nil {
				t.Fatalf("missing comparison report: %v\n%s", err, result.Text)
			}
			var report upstreamEvidence
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			wantExit := map[string]int{"pass": 0, "fail": 1, "timeout": 124}[mode]
			if report.Native.Exit != 0 || report.Gart.Exit != wantExit || report.Gart.Outcomes["TestPass"] != "pass" {
				t.Fatalf("incorrect process evidence: native=%+v Gart=%+v", report.Native, report.Gart)
			}
			if mode == "pass" {
				if len(report.Comparison.Passed) != 2 || len(report.Comparison.Problems) != 0 {
					t.Fatalf("incorrect successful comparison: %+v", report.Comparison)
				}
			} else if len(report.Comparison.Passed) != 0 || len(report.Comparison.Problems) == 0 {
				t.Fatalf("failed process counted as parity: %+v", report.Comparison)
			}
			if mode == "timeout" {
				data, err := os.ReadFile(filepath.Join(dir, "timeout-report.gart.log"))
				if err != nil || !strings.Contains(string(data), "=== RUN   TestWait") || report.Gart.Outcomes["TestWait"] != "" {
					t.Fatalf("lost partial output: %v\n%s", err, data)
				}
			}
		})
	}
}
