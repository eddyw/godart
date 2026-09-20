package main

import (
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTestCommand(t *testing.T) {
	if _, err := exec.LookPath("dart"); err != nil {
		t.Skip("the test requires Dart")
	}
	dir := t.TempDir()
	for name, text := range map[string]string{
		"go.mod": "module example.com/cliflags\n\ngo 1.27\n",
		"flags_test.go": `package cliflags
import ("flag"; "testing")
var enabled = flag.Bool("enabled", false, "")
var label = flag.String("label", "", "")
func TestFlags(t *testing.T) {
 if !flag.Parsed() || !*enabled || *label != "two words" || !testing.Short() {
  t.Fatalf("incorrect flags: %v %v %q %v", flag.Parsed(), *enabled, *label, testing.Short())
 }
 t.Log("custom flags reached the test")
}
func TestOther(t *testing.T) { t.Fatal("the filter must exclude this test") }
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(t.TempDir(), "gart")
	if text, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, text)
	}
	args := []string{".", "-run", "^TestFlags$", "-short", "-v", "-args", "-enabled", "-label", "two words"}
	for _, command := range [][]string{
		append([]string{"go", "test"}, args...),
		append([]string{binary, "test", "-frontend", "source", "-o", filepath.Join(t.TempDir(), "output")}, args...),
	} {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Dir = dir
		text, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", command[0], err, text)
		}
		if !strings.Contains(string(text), "custom flags reached the test") || strings.Contains(string(text), "TestOther") {
			t.Fatalf("unexpected output from %s:\n%s", command[0], text)
		}
	}
}

func TestTestArgsHelp(t *testing.T) {
	for _, arg := range []string{"-h", "-help", "--help"} {
		set := flag.NewFlagSet("test", flag.ContinueOnError)
		set.SetOutput(io.Discard)
		args, _, err := testArgs(set, []string{"./a", arg})
		if err != nil {
			t.Fatal(err)
		}
		if err := set.Parse(args); err != flag.ErrHelp {
			t.Fatalf("%s: got %v, want ErrHelp", arg, err)
		}
	}
}

func TestTestArgs(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		args, compiler, program []string
		wantError               bool
	}{
		{"after package", []string{"./a", "-run", "^TestOne$", "-v"}, []string{"-run", "^TestOne$", "-v", "./a"}, nil, false},
		{"mixed packages", []string{"-v", "./a", "-run=One", "./b", "-count", "2"}, []string{"-v", "-run=One", "./a"}, []string{"./b", "-count", "2"}, false},
		{"explicit false", []string{"./a", "--v=false"}, []string{"--v=false", "./a"}, nil, false},
		{"args", []string{"./a", "-v", "-args", "-label", "two words", "./literal", "-v=false"}, []string{"-v", "./a"}, []string{"-label", "two words", "./literal", "-v=false"}, false},
		{"double dash", []string{"--", "-label=x"}, nil, []string{"--", "-label=x"}, false},
		{"value starts with dash", []string{"./a", "-run", "-args"}, []string{"-run", "-args", "./a"}, nil, false},
		{"missing value", []string{"./a", "-run"}, nil, nil, true},
		{"unknown flag", []string{"./a", "-label=x"}, []string{"./a"}, []string{"-label=x"}, false},
		{"unknown value", []string{"./a", "-label", "two words", "-v"}, []string{"-v", "./a"}, []string{"-label", "two words"}, false},
		{"contiguous packages", []string{"./a", "./b", "-v"}, []string{"-v", "./a", "./b"}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := flag.NewFlagSet("test", flag.ContinueOnError)
			set.Bool("v", false, "")
			set.String("run", "", "")
			set.Int("count", 1, "")
			compiler, program, err := testArgs(set, tc.args)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v", err)
			}
			if !reflect.DeepEqual(compiler, tc.compiler) || !reflect.DeepEqual(program, tc.program) {
				t.Fatalf("compiler = %q, program = %q", compiler, program)
			}
			if err == nil {
				if err := set.Parse(compiler); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
