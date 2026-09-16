package driver_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

// TestWebFetch proves that the web target reaches a real server. A browser
// holds no socket, thus net/http rides the fetch API of the host there. The
// test runs the same program with the go command and with the 2 web builds,
// and each output must match. Read decision 252.
func TestWebFetch(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("the machine holds no node: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(webFetchRoutes))
	defer server.Close()

	source, err := filepath.Abs("../../testdata/web/webfetch")
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	program := t.TempDir()
	for _, name := range []string{"go.mod", "main.go"} {
		text, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		body := strings.Replace(string(text),
			`"http://127.0.0.1:1"`, quoted(server.URL), 1)
		if err := os.WriteFile(filepath.Join(program, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want := runProgram(t, program, "go", "run", ".")

	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Trace:    traceBuild(t),
		Dir:      program,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtimeDir,
		Format:   true,
	}); err != nil {
		t.Fatal(err)
	}
	run(t, out, "dart", "pub", "get")

	run(t, out, "dart", "compile", "js", "bin/main.dart", "-o", "main.js")
	if got := runNodeJS(t, out); got != want {
		t.Errorf("the output of the JS build differs\nwant:\n%s\ngot:\n%s", want, got)
	}

	run(t, out, "dart", "compile", "wasm", "bin/main.dart", "-o", "main.wasm")
	runner := `import { compile } from './main.mjs';
import { readFile } from 'node:fs/promises';

const bytes = await readFile(new URL('./main.wasm', import.meta.url));
const app = await compile(bytes);
const instance = await app.instantiate({});
instance.invokeMain();
`
	if err := os.WriteFile(filepath.Join(out, "run.mjs"), []byte(runner), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := runProgram(t, out, "node", "run.mjs"); got != want {
		t.Errorf("the output of the wasm build differs\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// quoted gives the Go text of the address of the server.
func quoted(url string) string { return `"` + url + `"` }

// webFetchRoutes answers the 4 requests of the program. The body of /big needs
// more than 1 read of the stream, thus it proves the read of a chunk.
func webFetchRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/hello":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "hello from the server")
	case "/echo":
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%s %s %s", r.Method, r.Header.Get("X-Gart"), body)
	case "/big":
		w.Header().Set("Content-Type", "application/octet-stream")
		block := make([]byte, 1000)
		for i := range block {
			block[i] = byte(i % 251)
		}
		for i := 0; i < 100; i++ {
			w.Write(block)
		}
	case "/slow":
		time.Sleep(2 * time.Second)
		io.WriteString(w, "late")
	default:
		http.NotFound(w, r)
	}
}
