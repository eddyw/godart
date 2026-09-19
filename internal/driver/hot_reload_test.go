package driver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStringLiteralHotReload(t *testing.T) {
	if _, err := exec.LookPath("dart"); err != nil {
		t.Skip("Dart SDK unavailable")
	}
	dir, out := t.TempDir(), t.TempDir()
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	putKernelFile(t, filepath.Join(dir, "go.mod"), "module example.com/reloadprobe\n\ngo 1.27.0\n")
	build := func(value string) {
		t.Helper()
		putKernelFile(t, filepath.Join(dir, "app.go"), fmt.Sprintf(`package app
var Count int
func Value() string { Count++; return %q }
`, value))
		if err := Build(Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtime}); err != nil {
			t.Fatal(err)
		}
	}
	build("first")
	if err := pubGet(out, runtime); err != nil {
		t.Fatal(err)
	}
	putKernelFile(t, filepath.Join(out, "probe.dart"), `import 'dart:convert';
import 'dart:io';
import 'package:reloadprobe/app.dart' as app;
void report() { print('probe: ${app.Value().toDartString()}:${app.Count}'); }
Future<void> main() async {
  app.$init();
  report();
  await for (final _ in stdin.transform(utf8.decoder).transform(const LineSplitter())) { report(); }
}
`)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	infoPath := filepath.Join(out, "service.json")
	cmd := exec.CommandContext(ctx, "dart", "--enable-vm-service=0", "--write-service-info="+infoPath, "run", "probe.dart")
	cmd.Dir = out
	cmd.Stderr = os.Stderr
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { input.Close(); cmd.Process.Kill(); cmd.Wait() }()
	lines := make(chan string, 16)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "probe:") {
				lines <- scanner.Text()
			}
		}
	}()
	wantLine := func(want string) {
		t.Helper()
		select {
		case got := <-lines:
			if got != want {
				t.Fatalf("output %q, want %q", got, want)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	wantLine("probe: first:1")
	info, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatal(err)
	}
	var service struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(info, &service); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	request := func(method string, result any) {
		t.Helper()
		response, err := client.Get(service.URI + method)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var envelope struct {
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if len(envelope.Error) != 0 {
			t.Fatalf("VM service: %s", envelope.Error)
		}
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			t.Fatal(err)
		}
	}
	var vm struct {
		Isolates []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"isolates"`
	}
	request("getVM", &vm)
	isolate := ""
	for _, item := range vm.Isolates {
		if item.Name == "main" {
			isolate = item.ID
		}
	}
	if isolate == "" {
		t.Fatal("main isolate unavailable")
	}
	for index, value := range []string{"second", "first"} {
		build(value)
		var report struct {
			Success bool `json:"success"`
		}
		request("reloadSources?isolateId="+url.QueryEscape(isolate), &report)
		if !report.Success {
			t.Fatal("VM rejected reload")
		}
		if _, err := fmt.Fprintln(input); err != nil {
			t.Fatal(err)
		}
		wantLine(fmt.Sprintf("probe: %s:%d", value, index+2))
	}
}
