package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostInputsTrackDependencies(t *testing.T) {
	root := t.TempDir()
	a := &app{dir: filepath.Join(root, "app"), root: root, out: "gen", host: "lib/widgets_gen.dart"}
	write := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"app/.dart_tool/package_config.json": `{"configVersion":2,"packages":[{"rootUri":"../"},{"rootUri":"../gen"},{"rootUri":"../../dependency"}]}`,
		"app/pubspec.yaml":                   "name: fixture\n", "app/pubspec.lock": "version: 1\n",
		"app/lib/brand.dart": "class Brand {}\n", "app/gen/lib/app.dart": "generated\n",
		"app/lib/widgets_gen.dart": "host\n", "dependency/lib/widget.dart": "class Widget {}\n",
		"dependency/LICENSE": "dependency license\n", "flutter/nodegen/pubspec.lock": "version: 1\n",
		"flutter/nodegen/lib/read.dart": "generator\n", "app/glutter.bindings.json": "{}\n",
	}
	for path, text := range files {
		write(filepath.Join(root, path), text)
	}
	digest := func() string {
		t.Helper()
		key, err := a.hostInputs()
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	previous := digest()
	for _, path := range []string{"app/gen/lib/app.dart", "app/lib/widgets_gen.dart"} {
		write(filepath.Join(root, path), "new generated output\n")
		if got := digest(); got != previous {
			t.Fatalf("output %s invalidated host inputs", path)
		}
	}
	for _, path := range []string{"app/lib/brand.dart", "dependency/lib/widget.dart", "dependency/LICENSE", "app/pubspec.lock", "app/glutter.bindings.json", "flutter/nodegen/lib/read.dart"} {
		write(filepath.Join(root, path), files[path]+"changed\n")
		got := digest()
		if got == previous {
			t.Fatalf("input %s did not invalidate the host", path)
		}
		previous = got
	}
}
