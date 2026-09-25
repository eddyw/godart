package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRunStreams(t *testing.T) {
	dir := t.TempDir()
	for name, text := range map[string]string{
		"go.mod": "module example.com/runstreams\n\ngo 1.27\n",
		"main.go": `package main
import ("fmt"; "os")
type text string
func show[T ~string](s T) { print(s); println(s) }
func main() {
 fmt.Println("stdout")
 fmt.Fprintln(os.Stderr,"stderr")
 print("builtin print")
 println(" line")
 raw := make([]byte,256)
 for i := range raw { raw[i]=byte(i) }
 s := string(raw)+"€𐀀"
 print(s); println(text(s))
 show(s); show(text(s))
 os.Stdout.Write([]byte{0xff,0,10})
 os.Stderr.Write([]byte{0xfe,0,10})
 if len(os.Args)>1 { os.Exit(7) }
}
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cli := filepath.Join(t.TempDir(), "gart")
	native := filepath.Join(t.TempDir(), "program")
	for _, command := range []struct {
		dir  string
		args []string
	}{
		{".", []string{"build", "-o", cli, "."}},
		{dir, []string{"build", "-o", native, "."}},
	} {
		cmd := exec.Command("go", command.args...)
		cmd.Dir = command.dir
		if text, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build: %v\n%s", err, text)
		}
	}
	type result struct {
		stdout, stderr string
		exit           int
	}
	capture := func(t *testing.T, name string, args ...string) result {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if ctx.Err() != nil || !errors.As(err, &exit) {
				t.Fatalf("execute %s: %v\n%s", name, err, stderr.String())
			}
			code = exit.ExitCode()
		}
		return result{stdout.String(), stderr.String(), code}
	}
	for _, frontend := range []string{"auto", "source", "resident"} {
		t.Run(frontend, func(t *testing.T) {
			out := t.TempDir()
			for _, failed := range []bool{false, true} {
				var program []string
				wantExit := 0
				if failed {
					program = []string{"fail"}
					wantExit = 7
				}
				want := capture(t, native, program...)
				if want.exit != wantExit {
					t.Fatalf("native exit = %d, want %d", want.exit, wantExit)
				}
				args := []string{"run", "-frontend", frontend, "-format=false", "-o", out, ".", "--"}
				got := capture(t, cli, append(args, program...)...)
				if got != want {
					t.Fatalf("Go: exit=%d stdout=%q stderr=%q\nGart: exit=%d stdout=%q stderr=%q", want.exit, want.stdout, want.stderr, got.exit, got.stdout, got.stderr)
				}
				t.Logf("matching exit=%d stdout_bytes=%d stderr_bytes=%d", got.exit, len(got.stdout), len(got.stderr))
			}
		})
	}
}
