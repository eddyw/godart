package driver

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func kernelFixture(t *testing.T) (Options, *int) {
	t.Helper()
	if _, err := exec.LookPath("dart"); err != nil {
		t.Skip("Dart SDK unavailable")
	}
	out := filepath.Join(t.TempDir(), "space $cash #hash")
	putKernelFile(t, filepath.Join(out, ".dart_tool", "package_config.json"), `{"configVersion":2,"packages":[]}`)
	putKernelFile(t, filepath.Join(out, "dep.dart"), `const value = 'first';`)
	putKernelFile(t, filepath.Join(out, "main.dart"), `import 'dep.dart'; void main() {print(value);}`)
	count := new(int)
	return Options{Out: out, Trace: func(phase string, _ time.Duration) {
		if phase == "kernel-compile" {
			*count++
		}
	}}, count
}

func putKernelFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestKernelCacheInvalidation(t *testing.T) {
	opt, count := kernelFixture(t)
	build := func(want string, compiles int) string {
		t.Helper()
		path, err := opt.kernel("main.dart")
		if err != nil {
			t.Fatal(err)
		}
		text, code, err := dartOutput(opt.Out, []string{"run", path})
		if err != nil || code != 0 || text != want {
			t.Fatalf("output %q, exit %d, error %v", text, code, err)
		}
		if *count != compiles {
			t.Fatalf("compiles %d, want %d", *count, compiles)
		}
		return path
	}
	path := build("first\n", 1)
	build("first\n", 1)
	dep := filepath.Join(opt.Out, "dep.dart")
	old, err := os.Stat(dep)
	if err != nil {
		t.Fatal(err)
	}
	putKernelFile(t, dep, `const value = 'other';`)
	if err := os.Chtimes(dep, old.ModTime(), old.ModTime()); err != nil {
		t.Fatal(err)
	}
	build("other\n", 2)
	putKernelFile(t, filepath.Join(opt.Out, "next.dart"), `const next = 'third';`)
	putKernelFile(t, dep, `import 'next.dart'; const value = next;`)
	build("third\n", 3)
	putKernelFile(t, filepath.Join(opt.Out, "next.dart"), `const next = 'fourth';`)
	build("fourth\n", 4)
	putKernelFile(t, filepath.Join(opt.Out, ".dart_tool", "package_config.json"), `{"configVersion":2,"packages":[],"generator":"changed"}`)
	build("fourth\n", 5)
	putKernelFile(t, path, "corrupt")
	build("fourth\n", 6)
	manifest := strings.TrimSuffix(path, ".dill") + ".json"
	putKernelFile(t, manifest, `{}`)
	build("fourth\n", 7)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	build("fourth\n", 8)
	opt.NoCache = true
	build("fourth\n", 9)
	opt.NoCache = false
	if err := os.Remove(filepath.Join(opt.Out, "next.dart")); err != nil {
		t.Fatal(err)
	}
	if _, err := opt.kernel("main.dart"); err == nil {
		t.Fatal("missing dependency reused a kernel")
	}
}

func TestKernelExecutionContract(t *testing.T) {
	opt, _ := kernelFixture(t)
	main := filepath.Join(opt.Out, "main.dart")
	putKernelFile(t, main, `import 'dart:io';
void main(List<String> args) {
 var asserts = false;
 assert(() {asserts = true; return true;}());
 print('product=${const bool.fromEnvironment("dart.vm.product")} asserts=$asserts');
 print('${args.join("|")}:${stdin.readLineSync()}:${Directory.current.path}');
 stderr.writeln('error-channel');
 exit(7);
}`)
	kernel, err := opt.kernel("main.dart")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source, err := os.ReadFile(main)
	if err != nil {
		t.Fatal(err)
	}
	reference := filepath.Join(t.TempDir(), "main.dart")
	putKernelFile(t, reference, string(source))
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, asserts := range []bool{false, true} {
		execute := func(entry string) (string, string, int) {
			args := []string{"run"}
			if asserts {
				args = append(args, "--enable-asserts")
			}
			args = append(args, entry, "two words", "--literal")
			cmd := exec.Command("dart", args...)
			cmd.Dir = dir
			cmd.Stdin = strings.NewReader("input\n")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			code := exitCode(err)
			if code != 7 {
				t.Fatalf("entry %s, exit %d, err %v, stdout %q, stderr %q", entry, code, err, stdout.String(), stderr.String())
			}
			return stdout.String(), stderr.String(), code
		}
		a, b, c := execute(reference)
		x, y, z := execute(kernel)
		if a != x || b != y || c != z {
			t.Fatalf("source %q %q %d; kernel %q %q %d", a, b, c, x, y, z)
		}
		if !strings.Contains(x, "product=false") || !strings.Contains(x, "two words|--literal:input:"+dir) || y != "error-channel\n" {
			t.Fatalf("unexpected output %q %q", x, y)
		}
	}
}

func TestKernelSDKIdentity(t *testing.T) {
	opt, count := kernelFixture(t)
	real, err := exec.LookPath("dart")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	version := filepath.Join(bin, "version.txt")
	putKernelFile(t, version, "first")
	t.Setenv("KERNEL_TEST_DART", real)
	t.Setenv("KERNEL_TEST_VERSION", version)
	wrapper := "#!/bin/sh\nif [ \"$1\" = --version ]; then cat \"$KERNEL_TEST_VERSION\"; else exec \"$KERNEL_TEST_DART\" \"$@\"; fi\n"
	putKernelFile(t, filepath.Join(bin, "dart"), wrapper)
	if err := os.Chmod(filepath.Join(bin, "dart"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := opt.kernel("main.dart"); err != nil {
		t.Fatal(err)
	}
	if _, err := opt.kernel("main.dart"); err != nil {
		t.Fatal(err)
	}
	if *count != 1 {
		t.Fatal("SDK identity missed an unchanged cache")
	}
	putKernelFile(t, version, "second")
	if _, err := opt.kernel("main.dart"); err != nil {
		t.Fatal(err)
	}
	if *count != 2 {
		t.Fatal("SDK change reused a kernel")
	}
}

func TestKernelDependencies(t *testing.T) {
	cases := map[string][]string{
		"out: /one /two\n": {"/one", "/two"},
		"out\\ file: /one\\ two /hash\\# /cash$$ /slash\\\\end\n": {"/one two", "/hash#", "/cash$", "/slash\\end"},
		"out: /one \\\n /two": {"/one", "/two"},
	}
	for input, want := range cases {
		got, err := kernelDependencies(input)
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("%q: %v %v, want %v", input, got, err, want)
		}
	}
	for _, input := range []string{"", "out:", "no separator"} {
		if _, err := kernelDependencies(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestKernelReceiptIncludesImportedSources(t *testing.T) {
	opt, _ := kernelFixture(t)
	path, err := opt.kernel("main.dart")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(strings.TrimSuffix(path, ".dill") + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt kernelReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.dart", "dep.dart", ".dart_tool/package_config.json"} {
		if receipt.Dependencies[filepath.Join(opt.Out, name)] == "" {
			t.Errorf("missing dependency %s", name)
		}
	}
}

func TestKernelEditDuringCompilation(t *testing.T) {
	opt, count := kernelFixture(t)
	real, err := exec.LookPath("dart")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	t.Setenv("KERNEL_TEST_DART", real)
	t.Setenv("KERNEL_TEST_DEP", filepath.Join(opt.Out, "dep.dart"))
	wrapper := `#!/bin/sh
"$KERNEL_TEST_DART" "$@" || exit $?
if [ "$1" = compile ]; then
 printf "const value = 'other';" > "$KERNEL_TEST_DEP"
fi
`
	putKernelFile(t, filepath.Join(bin, "dart"), wrapper)
	if err := os.Chmod(filepath.Join(bin, "dart"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := opt.kernel("main.dart"); err != nil {
		t.Fatal(err)
	}
	kernel, err := opt.kernel("main.dart")
	if err != nil {
		t.Fatal(err)
	}
	output, code, err := dartOutput(opt.Out, []string{"run", kernel})
	if err != nil || code != 0 || output != "other\n" || *count != 2 {
		t.Fatalf("output %q, code %d, error %v, compiles %d", output, code, err, *count)
	}
}
