package driver_test

import (
	"archive/zip"
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestEmbedModuleCache(t *testing.T) {
	const module = "example.com/assets"
	const version = "v1.0.0"
	proxy := t.TempDir()
	cache := t.TempDir()
	t.Cleanup(func() {
		if err := filepath.WalkDir(cache, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return nil
		}); err != nil {
			t.Error(err)
		}
	})
	endpoint := filepath.Join(proxy, "example.com", "assets", "@v")
	if err := os.MkdirAll(endpoint, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod":  "module " + module + "\n\ngo 1.27\n",
		"LICENSE": "Copyright 2026 Gart contributors. Regression fixture.\n",
		"assets.go": `package assets

import (
	"embed"
	"strings"
)

//go:embed version.txt
var text string

var Version = strings.TrimSpace(text)

//go:embed data.bin
var Bytes []byte

//go:embed static
var Files embed.FS
`,
		"version.txt":  "1.2.3\n",
		"data.bin":     "\x00\xff\x80\x7f",
		"static/a.txt": "alpha\n",
		"static/b.txt": "beta\n",
	}
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for name, content := range files {
		w, err := zw.Create(module + "@" + version + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(endpoint, version+".zip"), archive.String())
	writeFile(t, filepath.Join(endpoint, version+".mod"), files["go.mod"])
	writeFile(t, filepath.Join(endpoint, version+".info"), `{"Version":"v1.0.0","Time":"2026-01-01T00:00:00Z"}`)
	proxyURL := url.URL{Scheme: "file", Path: filepath.ToSlash(proxy)}
	t.Setenv("GOPROXY", proxyURL.String())
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOMODCACHE", cache)
	t.Setenv("GOWORK", "off")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/embedtest\n\ngo 1.27\n\nrequire "+module+" "+version+"\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"example.com/assets"
	"fmt"
)

func report() string {
	entries, err := assets.Files.ReadDir("static")
	if err != nil { panic(err) }
	result := fmt.Sprintf("%s %x\n", assets.Version, assets.Bytes)
	for _, entry := range entries {
		data, err := assets.Files.ReadFile("static/" + entry.Name())
		if err != nil { panic(err) }
		result += entry.Name() + ":" + string(data)
	}
	return result
}

func main() { fmt.Print(report()) }
`)
	writeFile(t, filepath.Join(dir, "main_test.go"), `package main

import "testing"

func TestAssets(t *testing.T) {
	const want = "1.2.3 00ff807f\na.txt:alpha\nb.txt:beta\n"
	if got := report(); got != want { t.Fatalf("got %q, want %q", got, want) }
}
`)
	run(t, dir, "go", "mod", "download", module)
	want := runProgram(t, dir, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t),
	}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	if got := runProgram(t, out, "dart", "--packages=.dart_tool/package_config.json", "bin/main.dart"); got != want {
		t.Fatalf("Go: %q\nGart: %q", want, got)
	}
	if got, want := gartTestIn(t, dir, t.TempDir(), driver.TestOptions{}), goTest(t, dir, false); got != want {
		t.Fatalf("Go tests: %q\nGart tests: %q", want, got)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(cache, module+"@"+version, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("cached dependency file %s changed", name)
		}
	}
}

// buildEmbed compiles a module of 1 file and gives the report of the build.
func buildEmbed(t *testing.T, main string, files map[string]string) error {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module embedtest\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), main)
	for name, text := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
			t.Fatal(err)
		}
		writeFile(t, full, text)
	}
	return driver.Build(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
	})
}

// TestEmbedRejectsTheConstructor proves that a file of the user cannot name
// the constructor of the package embed. The real package of Go holds no such
// name, thus a call would compile here and fail under the go command.
func TestEmbedRejectsTheConstructor(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	"embed"
	"fmt"
)

func main() {
	f := embed.New([]string{"a.txt"}, []string{"a"})
	text, _ := f.ReadFile("a.txt")
	fmt.Println(string(text))
}
`, nil)
	if err == nil {
		t.Fatal("the build accepted a call of embed.New")
	}
	want := "the package embed of Go holds no such name"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// TestEmbedRejectsAPatternThatNamesNoFile follows the go command, which
// rejects a directive whose pattern reaches nothing.
func TestEmbedRejectsAPatternThatNamesNoFile(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	"embed"
	"fmt"
)

//go:embed static/*.txt
var assets embed.FS

func main() { fmt.Println(assets) }
`, map[string]string{"static/keep.bin": "x"})
	if err == nil {
		t.Fatal("the build accepted a pattern that names no file")
	}
	want := "the pattern static/*.txt names no file"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// TestEmbedRejectsAPatternOutsideThePackage keeps a directive inside the
// directory of its package.
func TestEmbedRejectsAPatternOutsideThePackage(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	"embed"
	"fmt"
)

//go:embed ../secret.txt
var assets embed.FS

func main() { fmt.Println(assets) }
`, nil)
	if err == nil {
		t.Fatal("the build accepted a pattern outside the package")
	}
	want := "the pattern ../secret.txt is outside the rules"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// TestEmbedRejects2FilesForAString follows the go command, which gives a
// string or a byte slice the content of 1 file.
func TestEmbedRejects2FilesForAString(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	_ "embed"
	"fmt"
)

//go:embed static
var text string

func main() { fmt.Println(text) }
`, map[string]string{"static/a.txt": "a", "static/b.txt": "b"})
	if err == nil {
		t.Fatal("the build accepted 2 files for a string")
	}
	want := "takes 1 file, and the patterns name 2"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}
