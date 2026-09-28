package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDirectoryMetadata(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("directory metadata validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/directorymetadata\n\ngo 1.27\n")
	osDir := filepath.Join(runtime.GOROOT(), "src/os")
	writeFile(t, filepath.Join(dir, "os_test.go"), selectedUpstreamSource(t, filepath.Join(osDir, "os_test.go"), []string{"dot", "sysDir", "sysdir", "equal", "testReaddirnames", "testReaddir", "testReadDir", "TestFileReaddirnames", "TestFileReaddir", "TestFileReadDir", "TestDirSeek"}))

	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	const filter = "^(TestFileReaddirnames|TestFileReaddir|TestFileReadDir|TestDirSeek)$"
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "test", "-c", "-o", native, ".")
	t.Log(run(t, osDir, native, "-test.v", "-test.short="+strconv.FormatBool(testing.Short()), "-test.run="+filter))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = programOutcome(t, osDir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, osDir, args)
			}
			compareUpstream(t, "os-directory-metadata", "os", filter, target, osDir, result.Text, result.Exit)
		})
	}
}

func TestDirectoryMetadataAOT(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("directory metadata validation requires macOS or Linux")
	}
	dir, err := filepath.Abs("../../testdata/programs/directorymetadata")
	if err != nil {
		t.Fatal(err)
	}
	want := runProgram(t, dir, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	exe := filepath.Join(out, "program")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
	if got := runProgram(t, out, exe); got != want {
		t.Fatalf("directory metadata differs\nGo:\n%s\nGart:\n%s", want, got)
	}
}
