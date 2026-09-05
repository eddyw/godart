package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// newApp writes a new application that Go drives. The skeleton holds the Go
// module, the Dart host, and the pubspec, thus 1 command reaches a screen.
func newApp(args []string) error {
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: glutter new <name>")
	}
	name := args[0]
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("%s stands already", name)
	}
	here, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := filepath.Join(here, name)
	root, err := findRoot(here)
	if err != nil {
		return err
	}
	rel := func(to string) string {
		p, err := filepath.Rel(filepath.Join(dir, "go"), filepath.Join(root, to))
		if err != nil {
			return filepath.Join(root, to)
		}
		return p
	}
	relDart := func(to string) string {
		p, err := filepath.Rel(dir, filepath.Join(root, to))
		if err != nil {
			return filepath.Join(root, to)
		}
		return p
	}
	title := strings.ToUpper(name[:1]) + name[1:]
	// The Dart package of the application and the Dart package of the Go
	// output stand in the same resolution, thus the 2 names must differ.
	appName := name + "_app"
	goName := "go" + name

	files := map[string]string{
		"pubspec.yaml": fmt.Sprintf(pubspecTemplate, appName,
			relDart("flutter/goflutter"), relDart("runtime/goruntime"), goName),
		"analysis_options.yaml": analysisTemplate,
		"go/go.mod":             fmt.Sprintf(goModTemplate, goName, rel("flutter/ui")),
		"go/app.go":             appTemplate,
		"lib/main.dart":         fmt.Sprintf(mainTemplate, name, title, title, title, title),
		"lib/bridge.dart":       fmt.Sprintf(bridgeTemplate, goName, title),
		".gitignore":            gitignoreTemplate,
	}
	for path, body := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("%s holds a new application.\n\n", name)
	fmt.Printf("\tcd %s\n\tglutter run -d macos\n", name)
	return nil
}

const pubspecTemplate = `name: %[1]s
description: An application whose widget tree comes from Go.
publish_to: none
version: 0.1.0

environment:
  sdk: ^3.13.0

dependencies:
  flutter:
    sdk: flutter
  material_ui: ^1.1.1
  goflutter:
    path: %[2]s
  %[4]s:
    path: gen
  goruntime:
    path: %[3]s

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^6.0.0

flutter:
  uses-material-design: true
`

const analysisTemplate = `analyzer:
  exclude:
    - build/**
    - android/**
    - ios/**
    - web/**
    - windows/**
    - macos/**
    - linux/**
include: package:flutter_lints/flutter.yaml
`

const goModTemplate = `module example.dev/%s

go 1.27.0

require gart.dev/ui v0.0.0

replace gart.dev/ui => %s
`

const appTemplate = `// Package app holds the whole application. Each screen is a Go value, and the
// Dart host builds a real Flutter widget from it.
package app

import (
	"strconv"

	"gart.dev/ui"
	"gart.dev/ui/material"
)

var count ui.Signal[int]

// Tree gives the widget tree of the application. The host calls it.
func Tree() ui.Node {
	return material.Scaffold{
		AppBar: material.AppBar{Title: ui.Text{Data: "This is Go"}},
		Body: ui.Center{Child: ui.Watch{Build: body}},
		FloatingActionButton: material.FloatingActionButton{
			Child:     ui.Icon{Icon: "add"},
			OnPressed: func() { count.Set(count.Peek() + 1) },
		},
	}
}

func body() ui.Node {
	return ui.Text{
		Data:  strconv.Itoa(count.Get()),
		Style: &ui.TextStyle{Size: ui.P(48.0)},
	}
}
`

const mainTemplate = `import 'package:material_ui/material_ui.dart';
import 'package:goflutter/goflutter.dart';

import 'bridge.dart';
import 'widgets_gen.dart';

void main() {
  goSetWidgets(goGenWidget);
  runApp(const %[2]sApp());
}

/// %[3]sApp holds the Material theme. Each widget under it comes from Go.
class %[4]sApp extends StatelessWidget {
  const %[5]sApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '%[1]s',
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        colorSchemeSeed: const Color(0xFF00ADD8),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: GoApp(bridge: const %[3]sBridge()),
    );
  }
}
`

const bridgeTemplate = `import 'package:%[1]s/app.dart' as app;
import 'package:%[1]s/dep/gart.dev/ui.dart' as ui;
import 'package:goflutter/goflutter.dart';
import 'package:goruntime/goruntime.dart';

/// %[2]sBridge forwards each call of the host into the Dart output of the Go
/// module.
final class %[2]sBridge implements GoBridge {
  const %[2]sBridge();

  @override
  void init() => app.$init();

  @override
  Iface? build() => app.Tree();

  @override
  int newWatch() => $ix(ui.NewWatch());

  @override
  Iface? runWatch(int id, Object? build) =>
      ui.RunWatch($gi(id), build as Iface? Function()?);

  @override
  void disposeWatch(int id) => ui.DisposeWatch($gi(id));

  @override
  List<int> takeDirty() {
    final Slice<GoInt> ids = ui.TakeDirty();
    return <int>[for (int i = 0; i < ids.len; i++) $ix(ids[i])];
  }

  @override
  int pump() => $ix(ui.Pump());

  @override
  void setTheme({
    required bool dark,
    required int primary,
    required int surface,
    required int onSurface,
    required double textSize,
  }) {
    ui.SetTheme(
      ui.ThemeData$(dark, $gi(primary), $gi(surface), $gi(onSurface), textSize),
    );
  }

  @override
  void setMedia({
    required double width,
    required double height,
    required double textScale,
    required double topInset,
    required double bottomInset,
  }) {
    ui.SetMedia(
      ui.MediaData$(width, height, textScale, topInset, bottomInset),
    );
  }

  @override
  String takeCalls() => ui.TakeCalls().toDartString();

  @override
  void answer(int id, String value, String error) => ui.Answer(
    $gi(id),
    GoString.fromString(value),
    GoString.fromString(error),
  );
}
`

const gitignoreTemplate = `gen/
.dart_tool/
build/
lib/widgets_gen.dart
macos/
ios/
android/
linux/
windows/
web/
.metadata
*.iml
`
