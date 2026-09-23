package driver_test

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type resolverInput struct {
	Network, Address string
	IP               []byte
	Port             int
	Zone             string
}

type resolverCases struct {
	TCP, UDP, IP []resolverInput
}

type resolverInputs struct {
	Source          string            `json:"source"`
	SourceSHA256    string            `json:"source_sha256"`
	GeneratorSource string            `json:"generator_source"`
	GeneratorSHA256 string            `json:"generator_sha256"`
	Generator       languageExecution `json:"generator"`
	Output          string            `json:"output"`
	Inputs          resolverCases     `json:"inputs"`
	FixtureSource   string            `json:"fixture_source"`
	FixtureSHA256   string            `json:"fixture_sha256"`
}

func prepareResolverInputs(t *testing.T, sourcePath string) *resolverInputs {
	t.Helper()
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, sourcePath, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := os.ReadFile("testdata/resolver-inputs.txt")
	if err != nil {
		t.Fatal(err)
	}
	end := positions.Position(file.Name.End()).Offset
	inputs := &resolverInputs{Source: string(source), SourceSHA256: languageDigest(source)}
	inputs.GeneratorSource = string(source[:end]) + "\nimport gartjson \"encoding/json\"\n" + string(source[end:]) + string(suffix)
	inputs.GeneratorSHA256 = languageDigest([]byte(inputs.GeneratorSource))
	dir := t.TempDir()
	path := filepath.Join(dir, "main_test.go")
	writeFile(t, path, inputs.GeneratorSource)
	overlay, err := json.Marshal(map[string]map[string]string{"Replace": {sourcePath: path}})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "overlay.json"), string(overlay))
	args := []string{"test", "-overlay=" + filepath.Join(dir, "overlay.json"), "-count=1", "-run=^TestGartResolverInputs$", "-v", "."}
	inputs.Output = runLanguage(t, &inputs.Generator, filepath.Dir(sourcePath), upstreamLimit(), "go", args...)
	inputs.Inputs, err = parseResolverInputs(inputs.Output)
	if err != nil {
		t.Fatal(err)
	}
	header := string(source[:positions.Position(file.Package).Offset])
	inputs.FixtureSource = resolverInputsSource(header, inputs.Inputs)
	inputs.FixtureSHA256 = languageDigest([]byte(inputs.FixtureSource))
	return inputs
}

func parseResolverInputs(output string) (resolverCases, error) {
	var data []string
	for _, line := range strings.Split(output, "\n") {
		if value, ok := strings.CutPrefix(line, "GART_RESOLVER_INPUTS "); ok {
			data = append(data, value)
		}
	}
	var inputs resolverCases
	if len(data) != 1 {
		return inputs, fmt.Errorf("expected 1 resolver input record, got %d", len(data))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data[0]), &fields); err != nil {
		return inputs, err
	}
	if len(fields) != 3 || fields["TCP"] == nil || fields["UDP"] == nil || fields["IP"] == nil {
		return inputs, fmt.Errorf("resolver input groups changed")
	}
	decoder := json.NewDecoder(strings.NewReader(data[0]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&inputs); err != nil {
		return inputs, err
	}
	return inputs, nil
}

func resolverInputsSource(header string, inputs resolverCases) string {
	var source strings.Builder
	source.WriteString(header + "package net_test\nimport . \"net\"\nfunc init() {\n")
	for _, group := range []struct {
		name string
		rows []resolverInput
	}{{"TCP", inputs.TCP}, {"UDP", inputs.UDP}, {"IP", inputs.IP}} {
		for _, row := range group.rows {
			fmt.Fprintf(&source, "resolve%sAddrTests = append(resolve%sAddrTests, resolve%sAddrTest{%q, %q, &%sAddr{IP: IP(%#v), Zone: %q", group.name, group.name, group.name, row.Network, row.Address, group.name, row.IP, row.Zone)
			if group.name != "IP" {
				fmt.Fprintf(&source, ", Port: %d", row.Port)
			}
			source.WriteString("}, nil})\n")
		}
	}
	source.WriteString("}\n")
	return source.String()
}

func TestResolverInputRecords(t *testing.T) {
	for _, output := range []string{"", "GART_RESOLVER_INPUTS broken", "GART_RESOLVER_INPUTS {}", "GART_RESOLVER_INPUTS {}\nGART_RESOLVER_INPUTS {}"} {
		if _, err := parseResolverInputs(output); err == nil {
			t.Fatalf("accepted invalid input: %q", output)
		}
	}
	inputs, err := parseResolverInputs("GART_RESOLVER_INPUTS {\"TCP\":null,\"UDP\":null,\"IP\":[{\"Network\":\"ip6\",\"Address\":\"::1%lo\\\"0\",\"IP\":\"AAEC\",\"Zone\":\"lo\\\"0\"}]}\n")
	if err != nil || len(inputs.IP) != 1 || inputs.IP[0].Zone != "lo\"0" {
		t.Fatalf("incorrect inputs: %+v, %v", inputs, err)
	}
	source := resolverInputsSource("// Upstream notice.\n", inputs)
	if _, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(source, "// Upstream notice.\n") || !strings.Contains(source, `Zone: "lo\"0"`) {
		t.Fatalf("source lost the notice or zone: %s", source)
	}
}
