package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestUpstreamTargets(t *testing.T) {
	var cases []struct {
		Package string                       `json:"package"`
		Targets []string                     `json:"targets"`
		Run     string                       `json:"run"`
		Exclude map[string]map[string]string `json:"exclude"`
	}
	data, err := os.ReadFile("../parity/testdata/targets.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	var inventory parity.Inventory
	data, err = os.ReadFile("../parity/testdata/" + runtime.GOOS + "-" + runtime.GOARCH + ".json")
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
			stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true, Trace: traceBuild(t)})
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
					args, _ := json.Marshal([]string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter})
					writeFile(t, filepath.Join(out, "runner.dart"), "import '"+filepath.ToSlash(stats.Binaries[0].Entry)+"' as tests;\nvoid main() { tests.main("+strings.ReplaceAll(string(args), "$", `\$`)+"); }\n")

					var result programResult
					switch target {
					case "aot":
						exe := filepath.Join(out, "tests.exe")
						run(t, out, "dart", "compile", "exe", "runner.dart", "-o", exe)
						result = programOutcomeWithLimit(t, dir, upstreamLimit(), exe)
					case "js":
						run(t, out, "dart", "compile", "js", "runner.dart", "-o", "main.js")
						writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis;\nrequire('./main.js');\n")
						result = programOutcomeWithLimit(t, dir, upstreamLimit(), "node", filepath.Join(out, "run.js"))
					case "wasm":
						run(t, out, "dart", "compile", "wasm", "runner.dart", "-o", "main.wasm")
						writeFile(t, filepath.Join(out, "run.mjs"), `import {compile} from './main.mjs';
import {readFile} from 'node:fs/promises';
const app = await compile(await readFile(new URL('./main.wasm', import.meta.url)));
const instance = await app.instantiate({});
instance.invokeMain();
`)
						result = programOutcomeWithLimit(t, dir, upstreamLimit(), "node", filepath.Join(out, "run.mjs"))
					}
					compareUpstream(t, pkg, pkg, filter, target, dir, result.Text, result.Exit)

				})
			}
		})
	}
}
