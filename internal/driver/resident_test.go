package driver

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestResidentRemovedImportAndRepair(t *testing.T) {
	opt, _ := kernelFixture(t)
	modes := []string{}
	opt.Trace = func(phase string, _ time.Duration) {
		if phase == "kernel-cold" || phase == "kernel-incremental" {
			modes = append(modes, phase)
		}
	}
	first, err := opt.kernel("main.dart")
	if err != nil {
		t.Fatal(err)
	}
	receipt := strings.TrimSuffix(first, ".dill") + ".json"
	saved, _ := os.ReadFile(receipt)
	hash, _ := fileDigest(first)
	putKernelFile(t, filepath.Join(opt.Out, "dep.dart"), "const value = ;")
	if _, err := opt.kernel("main.dart"); err == nil {
		t.Fatal("invalid source compiled")
	}
	after, _ := os.ReadFile(receipt)
	digest, _ := fileDigest(first)
	if string(after) != string(saved) || digest != hash {
		t.Fatal("compile failure changed the valid cache")
	}
	putKernelFile(t, filepath.Join(opt.Out, "dep.dart"), "const value='fixed';")
	if _, err := opt.kernel("main.dart"); err != nil {
		t.Fatal(err)
	}
	if modes[len(modes)-1] != "kernel-cold" {
		t.Fatal(modes)
	}
	if err := os.Remove(filepath.Join(opt.Out, "dep.dart")); err != nil {
		t.Fatal(err)
	}
	putKernelFile(t, filepath.Join(opt.Out, "main.dart"), "void main(){print('removed');}")
	kernel, err := opt.kernel("main.dart")
	if err != nil {
		t.Fatal(err)
	}
	text, code, err := dartOutput(opt.Out, []string{"run", kernel})
	if err != nil || code != 0 || text != "removed\n" {
		t.Fatalf("%q %d %v", text, code, err)
	}
	if modes[len(modes)-1] != "kernel-incremental" {
		t.Fatal(modes)
	}
}

func TestResidentRuntimeInvalidation(t *testing.T) {
	opt, _ := kernelFixture(t)
	runtime := t.TempDir()
	alias := filepath.Join(t.TempDir(), "runtime")
	if err := os.Symlink(runtime, alias); err != nil {
		t.Fatal(err)
	}
	dep := filepath.Join(runtime, "lib", "runtime.dart")
	putKernelFile(t, dep, "const value='runtime-a';")
	config := fmt.Sprintf(`{"configVersion":2,"packages":[{"name":"runtime","rootUri":%q,"packageUri":"lib/"}]}`, sourceURI(alias).String())
	putKernelFile(t, filepath.Join(opt.Out, ".dart_tool", "package_config.json"), config)
	putKernelFile(t, filepath.Join(opt.Out, "main.dart"), "import 'package:runtime/runtime.dart'; void main(){print(value);}")
	for _, want := range []string{"runtime-a", "runtime-z"} {
		stamp, err := os.Stat(dep)
		if err != nil {
			t.Fatal(err)
		}
		putKernelFile(t, dep, fmt.Sprintf("const value='%s';", want))
		if err := os.Chtimes(dep, stamp.ModTime(), stamp.ModTime()); err != nil {
			t.Fatal(err)
		}
		kernel, err := opt.kernel("main.dart")
		if err != nil {
			t.Fatal(err)
		}
		text, code, err := dartOutput(opt.Out, []string{"run", kernel})
		if err != nil || code != 0 || text != want+"\n" {
			t.Fatalf("%q %d %v", text, code, err)
		}
	}
	replacement := t.TempDir()
	putKernelFile(t, filepath.Join(replacement, "lib", "runtime.dart"), "const value='runtime-b';")
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(replacement, alias); err != nil {
		t.Fatal(err)
	}
	kernel, err := opt.kernel("main.dart")
	if err != nil {
		t.Fatal(err)
	}
	text, code, err := dartOutput(opt.Out, []string{"run", kernel})
	if err != nil || code != 0 || text != "runtime-b\n" {
		t.Fatalf("retargeted runtime: %q %d %v", text, code, err)
	}

}

func TestResidentEditDuringCompilation(t *testing.T) {
	opt, _ := kernelFixture(t)
	done := make(chan error, 1)
	go func() { _, err := opt.kernel("main.dart"); done <- err }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		files, _ := filepath.Glob(filepath.Join(opt.Out, ".gart", "kernel", "compile-*.dill"))
		if len(files) > 0 {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("compile finished before mutation: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("compile did not start")
		}
		time.Sleep(time.Millisecond)
	}
	dep := filepath.Join(opt.Out, "dep.dart")
	stamp, err := os.Stat(dep)
	if err != nil {
		t.Fatal(err)
	}
	putKernelFile(t, dep, "const value = 'other';")
	if err := os.Chtimes(dep, stamp.ModTime(), stamp.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil || !strings.Contains(err.Error(), "inputs changed") {
		t.Fatalf("mutation: %v", err)
	}
	files, _ := filepath.Glob(filepath.Join(opt.Out, ".gart", "kernel", "*.json"))
	if len(files) != 0 {
		t.Fatal("published a receipt after concurrent edit")
	}
	kernel, err := opt.kernel("main.dart")
	if err != nil {
		t.Fatal(err)
	}
	text, code, err := dartOutput(opt.Out, []string{"run", kernel})
	if err != nil || code != 0 || text != "other\n" {
		t.Fatalf("%q %d %v", text, code, err)
	}
}

func TestOutputOwnership(t *testing.T) {
	opt, _ := kernelFixture(t)
	release, err := opt.lockOutput()
	if err != nil {
		t.Fatal(err)
	}
	nested, err := opt.lockOutput()
	if err != nil {
		t.Fatal(err)
	}
	nested()
	other := Options{Out: opt.Out}
	if _, err := BuildWithStats(other); err == nil || !strings.Contains(err.Error(), "output directory is in use") {
		t.Fatalf("concurrent build: %v", err)
	}
	release()
	again, err := other.lockOutput()
	if err != nil {
		t.Fatal(err)
	}
	again()
}

func TestResidentServiceLifecycle(t *testing.T) {
	sdk, _, err := dartIdentity()
	if err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp("", "gart-service-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	socket := filepath.Join(directory, "service.sock")
	child, err := startResident(directory, socket, sdk)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Process.Kill()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		if time.Now().After(deadline) {
			data, _ := os.ReadFile(filepath.Join(directory, "service.log"))
			t.Fatal(string(data))
		}
		time.Sleep(20 * time.Millisecond)
	}
	request := func(opt Options, identity string) residentResponse {
		t.Helper()
		output, err := filepath.EvalSymlinks(opt.Out)
		if err != nil {
			t.Fatal(err)
		}
		entry := filepath.Join(output, "main.dart")
		sources, err := residentInputs(output, entry)
		if err != nil {
			t.Fatal(err)
		}
		artifact := filepath.Join(output, ".gart", "kernel", "result.dill")
		if err := os.MkdirAll(filepath.Dir(artifact), 0755); err != nil {
			t.Fatal(err)
		}
		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(20 * time.Second))
		err = json.NewEncoder(conn).Encode(residentRequest{Entry: entry, Output: output, Artifact: artifact, Identity: identity, Sources: sources})
		if err != nil {
			t.Fatal(err)
		}
		var response residentResponse
		if err := json.NewDecoder(conn).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Error != "" {
			t.Fatal(response.Error)
		}
		return response
	}
	opt, _ := kernelFixture(t)
	for i := range 102 {
		putKernelFile(t, filepath.Join(opt.Out, "dep.dart"), fmt.Sprintf("const value='%03d';", i))
		got := request(opt, "first").Mode
		want := "incremental"
		if i == 0 || i == 101 {
			want = "cold"
		}
		if got != want {
			t.Fatalf("request %d: %s, want %s", i, got, want)
		}
	}
	// A third output evicts the least recently used compiler.
	for range 2 {
		other, _ := kernelFixture(t)
		if request(other, "first").Mode != "cold" {
			t.Fatal("new output reused a compiler")
		}
	}
	if request(opt, "first").Mode != "cold" {
		t.Fatal("eviction did not release the compiler")
	}
	if request(opt, "changed").Mode != "cold" {
		t.Fatal("identity change did not restart the compiler")
	}
	deadline = time.Now().Add(75 * time.Second)
	for {
		if _, err := os.Stat(socket); os.IsNotExist(err) && child.Process.Signal(syscall.Signal(0)) != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("service survived idle timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestResidentLongTemporaryDirectory(t *testing.T) {
	opt, _ := kernelFixture(t)
	directory := filepath.Join(t.TempDir(), strings.Repeat("long-directory-", 10))
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", directory)
	if _, err := opt.kernel("main.dart"); err != nil {
		t.Fatal(err)
	}
}
