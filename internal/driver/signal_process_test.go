package driver_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestSignalProcessesUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native signal validation requires macOS or Linux")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os/signal")
	names := []string{"TestNotifyContextNotifications", "checkNotifyContext", "ctxNotifyTimes", "settleTime", "fatalWaitingTime", "init", "TestNohup", "TestStop", "sendUncaughtSighup", "dieFromSighup", "waitSig", "waitSig1", "quiesce"}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		if filepath.Base(path) == "signal_test.go" {
			sources[path] = []byte(selectedUpstreamSource(t, path, names))
		} else {
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + f.Name.Name + "\n")
		}
	}
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			out := t.TempDir()
			stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Binaries) != 1 {
				t.Fatalf("got %d test binaries", len(stats.Binaries))
			}
			prepareDart(t, out)
			var launcher string
			if target == "vm" {
				launcher = prepareUpstreamSelfExec(t, out, stats.Binaries[0].Entry)
			} else {
				resolved, err := filepath.EvalSymlinks(out)
				if err != nil {
					t.Fatal(err)
				}
				launcher = filepath.Join(resolved, "signal.test")
				entry := filepath.Join(out, stats.Binaries[0].Entry)
				source, err := os.ReadFile(entry)
				if err != nil {
					t.Fatal(err)
				}
				const start = "_main();"
				if strings.Count(string(source), start) != 1 {
					t.Fatal("expected 1 generated test entry point")
				}
				name := strings.ReplaceAll(strconv.Quote(launcher), "$", `\$`)
				writeFile(t, entry, upstreamOSImport(t, out)+strings.Replace(string(source), start, "$upstreamOS.Args[0] = GoString.fromString("+name+");\n"+start, 1))
				run(t, out, "dart", "compile", "exe", entry, "-o", launcher)
			}
			for _, name := range []string{"TestNotifyContextNotifications", "TestNohup"} {
				t.Run(name, func(t *testing.T) {
					filter := "^" + name + "$"
					args := upstreamTestArgs("-test.run=" + filter)
					got := programOutcome(t, dir, launcher, args...)
					compareUpstream(t, "signal-process-"+name, "os/signal", filter, target, dir, got.Text, got.Exit)
				})
			}

		})
	}
}
