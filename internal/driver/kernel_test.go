package driver

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var receipt kernelReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	for path := range receipt.Dependencies {
		if strings.HasSuffix(path, "next.dart") {
			delete(receipt.Dependencies, path)
		}
	}
	data, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	putKernelFile(t, manifest, string(data))
	build("fourth\n", 9)
	opt.NoCache = true
	build("fourth\n", 10)
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
	sdk, err := dartSDK()
	if err != nil {
		t.Fatal(err)
	}
	fake := t.TempDir()
	if err := os.Symlink(filepath.Join(sdk, "bin"), filepath.Join(fake, "bin")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(sdk, "lib"), filepath.Join(fake, "lib")); err != nil {
		t.Fatal(err)
	}
	// A launcher at the copied SDK root preserves that root during path resolution.
	bin := filepath.Join(fake, "launcher")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KERNEL_TEST_DART", filepath.Join(sdk, "bin", "dart"))
	putKernelFile(t, filepath.Join(bin, "dart"), "#!/bin/sh\nexec \"$KERNEL_TEST_DART\" \"$@\"\n")
	if err := os.Chmod(filepath.Join(bin, "dart"), 0755); err != nil {
		t.Fatal(err)
	}
	putKernelFile(t, filepath.Join(fake, "version"), "3.13.3")
	putKernelFile(t, filepath.Join(fake, "revision"), "first")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, want := range []int{1, 1} {
		if _, err := opt.kernel("main.dart"); err != nil {
			t.Fatal(err)
		}
		if *count != want {
			t.Fatalf("compiles %d, want %d", *count, want)
		}
	}
	putKernelFile(t, filepath.Join(fake, "revision"), "second")
	if _, err := opt.kernel("main.dart"); err != nil {
		t.Fatal(err)
	}
	if *count != 2 {
		t.Fatal("SDK change reused a kernel")
	}
	putKernelFile(t, filepath.Join(fake, "version"), "3.14.0")
	if _, err := opt.kernel("main.dart"); err == nil || !strings.Contains(err.Error(), "requires Dart 3.13.3") {
		t.Fatalf("SDK rejection: %v", err)
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
		path, err := filepath.EvalSymlinks(filepath.Join(opt.Out, name))
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Dependencies[path] == "" {
			t.Errorf("missing dependency %s", name)
		}
	}
}
