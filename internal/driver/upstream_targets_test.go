package driver_test

import (
	"encoding/json"
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestUpstreamTargets(t *testing.T) {
	var cases []struct {
		Package      string                       `json:"package"`
		Targets      []string                     `json:"targets"`
		Run          string                       `json:"run"`
		Exclude      map[string]map[string]string `json:"exclude"`
		FullNodeArgs []string                     `json:"full_node_args"`
	}
	data, err := os.ReadFile("../parity/testdata/targets.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	var inventory parity.Inventory
	inventoryName, err := parity.InventoryName(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join("..", "parity", "testdata", inventoryName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(cases))
	for _, c := range cases {
		names = append(names, c.Package)
	}
	selected := testShard(t, names)
	for _, c := range cases {
		if !selected[c.Package] {
			continue
		}
		pkg := c.Package
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkg))
			out := t.TempDir()
			options := driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true, Trace: traceBuild(t)}
			importer := prepareImporterInputs(t, pkg)
			if importer != nil {
				options.TestSources = importer.TestSources
			}
			var env []string
			if importer != nil || pkg == "go/importer" || pkg == "go/internal/gccgoimporter" || pkg == "internal/types/errors" {
				env = append(os.Environ(), "GOROOT="+build.Default.GOROOT)
			}
			stats, err := driver.BuildWithStats(options)
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Binaries) != 1 {
				t.Fatalf("got %d test binaries", len(stats.Binaries))
			}
			prepareDart(t, out)

			for _, target := range c.Targets {
				t.Run(target, func(t *testing.T) {
					filter := c.Run
					if exclusions := c.Exclude[target]; len(exclusions) > 0 {
						var names []string
						for _, p := range inventory.Packages {
							if p.Path != pkg {
								continue
							}
							for _, test := range p.Tests {
								if test.Eligible && test.Kind != "Benchmark" && exclusions[test.Name] == "" {
									names = append(names, regexp.QuoteMeta(test.Name))
								}
							}
						}
						filter = "^(" + strings.Join(names, "|") + ")$"
						for name, reason := range exclusions {
							t.Logf("target exclusion %s: %s", name, reason)
						}
					}
					args := upstreamTestArgs("-test.run="+filter, "-test.skip="+upstreamSkip(t, pkg))
					var nodeArgs []string
					if !testing.Short() {
						nodeArgs = c.FullNodeArgs
					}
					result := runUpstreamTargetEnv(t, target, out, stats.Binaries[0].Entry, dir, args, env, nodeArgs...)
					compareUpstreamSDKInventory(t, pkg, pkg, filter, target, dir, result.Text, result.Exit, inventory, nil, nil, c.Exclude[target], importer, true, nil)

				})
			}
		})
	}
}

func runUpstreamTarget(t *testing.T, target, out, entry, dir string, args []string, nodeArgs ...string) programResult {
	t.Helper()
	return runUpstreamTargetEnv(t, target, out, entry, dir, args, nil, nodeArgs...)
}

func runUpstreamTargetEnv(t *testing.T, target, out, entry, dir string, args, env []string, nodeArgs ...string) programResult {
	t.Helper()
	encoded, _ := json.Marshal(args)
	writeFile(t, filepath.Join(out, "runner.dart"), "import '"+filepath.ToSlash(entry)+"' as tests;\nvoid main() { tests.main("+strings.ReplaceAll(string(encoded), "$", `\$`)+"); }\n")

	var result programResult
	switch target {
	case "aot":
		exe := filepath.Join(out, "tests.exe")
		run(t, out, "dart", "compile", "exe", "runner.dart", "-o", exe)
		result = programOutcomeWithLimitEnv(t, dir, env, upstreamLimit(), exe)
	case "js":
		run(t, out, "dart", "compile", "js", "runner.dart", "-o", "main.js")
		writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis;\nrequire('./main.js');\n")
		result = programOutcomeWithLimitEnv(t, dir, env, upstreamLimit(), "node", append(slices.Clone(nodeArgs), filepath.Join(out, "run.js"))...)
	case "wasm":
		compiled := programOutcomeWithLimit(t, out, 16*time.Minute, "dart", "compile", "wasm", "runner.dart", "-o", "main.wasm")
		if compiled.Exit != 0 {
			t.Fatalf("Wasm compilation failed:\n%s", compiled.Text)
		}
		writeFile(t, filepath.Join(out, "run.mjs"), `import {compile} from './main.mjs';
import {readFile} from 'node:fs/promises';
const app = await compile(await readFile(new URL('./main.wasm', import.meta.url)));
const instance = await app.instantiate({});
instance.invokeMain();
`)
		result = programOutcomeWithLimitEnv(t, dir, env, upstreamLimit(), "node", append(slices.Clone(nodeArgs), filepath.Join(out, "run.mjs"))...)
	}
	return result
}
