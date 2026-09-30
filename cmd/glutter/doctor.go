package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/sdk"
)

// stamp holds the versions that wrote 1 pack of the node set.
type stamp struct {
	name     string
	flutter  string
	packages map[string]string
}

// packStamps reads the stamp of each pack of the node module.
func packStamps(dir string) ([]stamp, error) {
	var out []stamp
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "widgets_gen.go" {
			return err
		}
		s, err := readStamp(path)
		if err != nil {
			return err
		}
		s.name = filepath.Base(filepath.Dir(path))
		out = append(out, s)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s holds no pack of node types", dir)
	}
	return out, nil
}

// doctor reports the Flutter of the machine against the node set of the
// application, and it names each difference.
//
// A Flutter that drops a parameter of a constructor gives a Go field that the
// host does not read, and no error stands. The stamp of the generated file
// thus decides whether the node set matches the machine.
func (a *app) doctor() error {
	fmt.Printf("Glutter %s, Gart %s.\n", version, driver.Version)
	fmt.Println("compiler build Go:", runtime.Version())
	goEnv, err := sdk.ReadGoEnvironment(a.modulePath())
	if err != nil {
		return err
	}
	fmt.Println("active Go:", goEnv.Version)
	fmt.Println("Go source root:", goEnv.Root)
	fmt.Println("module:", goEnv.Module)
	language := goEnv.ModuleVersion
	if language == "" {
		language = "not declared"
	}
	fmt.Println("module Go directive:", language)
	live, err := liveVersions()
	if err != nil {
		return err
	}
	fmt.Printf("The machine holds Flutter %s and Dart %s.\n",
		live.flutter, live.packages["dart"])

	dir, err := a.uiModule()
	if err != nil {
		return err
	}
	packs, err := packStamps(dir)
	if err != nil {
		return err
	}
	bad := 0
	for _, held := range packs {
		if minorOf(held.flutter) != minorOf(live.flutter) {
			bad++
			fmt.Printf("\nThe pack %s comes from Flutter %s, and the machine holds %s.\n",
				held.name, held.flutter, live.flutter)
			continue
		}
		fmt.Printf("The pack %s comes from Flutter %s", held.name, held.flutter)
		for name, want := range held.packages {
			fmt.Printf(" and %s %s", name, want)
		}
		fmt.Println(".")
		for name, want := range held.packages {
			got := a.lockVersion(name)
			if got == "" || minorOf(got) == minorOf(want) {
				continue
			}
			bad++
			fmt.Printf("The pack %s comes from %s %s, and the application holds %s.\n",
				held.name, name, want, got)
		}
	}
	if bad == 0 {
		fmt.Println("\nThe node set matches the machine.")
		return nil
	}
	return fmt.Errorf("%d version differences stand. Run nodegen go again to write the node set of this machine", bad)
}

// uiModule gives the directory of the node module of the application.
func (a *app) uiModule() (string, error) {
	out, err := output(a.modulePath(), "go", "list", "-m", "-f", "{{.Dir}}", "gart.dev/ui")
	if err != nil {
		return "", fmt.Errorf("the module gart.dev/ui stands outside %s", a.modulePath())
	}
	return strings.TrimSpace(out), nil
}

// liveVersions reads the version of the Flutter of the machine.
func liveVersions() (stamp, error) {
	out, err := output(".", "flutter", "--version", "--machine")
	if err != nil {
		return stamp{}, fmt.Errorf("the flutter command gave no version: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		return stamp{}, err
	}
	s := stamp{packages: map[string]string{}}
	if v, ok := m["frameworkVersion"].(string); ok {
		s.flutter = v
	}
	if v, ok := m["dartSdkVersion"].(string); ok {
		s.packages["dart"] = strings.Fields(v)[0]
	}
	return s, nil
}

// readStamp reads the line that nodegen wrote into the head of a pack.
func readStamp(path string) (stamp, error) {
	f, err := os.Open(path)
	if err != nil {
		return stamp{}, err
	}
	defer f.Close()
	s := stamp{packages: map[string]string{}}
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		if !strings.HasPrefix(line, "// nodegen:") {
			if strings.HasPrefix(line, "package ") {
				break
			}
			continue
		}
		for _, field := range strings.Fields(strings.TrimPrefix(line, "// nodegen:")) {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			if key == "flutter" {
				s.flutter = value
				continue
			}
			s.packages[key] = value
		}
	}
	if s.flutter == "" {
		return s, fmt.Errorf("%s holds no stamp of nodegen. Run nodegen go again", path)
	}
	return s, scan.Err()
}

// lockVersion reads the version of 1 Dart package of pubspec.lock.
func (a *app) lockVersion(name string) string {
	f, err := os.Open(filepath.Join(a.dir, "pubspec.lock"))
	if err != nil {
		return ""
	}
	defer f.Close()
	found := false
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		if strings.TrimSpace(line) == name+":" && strings.HasPrefix(line, "  ") {
			found = true
			continue
		}
		if !found {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && key == "version" {
			return strings.Trim(strings.TrimSpace(value), `"`)
		}
		if len(line) > 2 && line[2] != ' ' {
			return ""
		}
	}
	return ""
}

// minorOf gives the major and the minor of a version, for example 3.47.
func minorOf(v string) string {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return v
	}
	return parts[0] + "." + parts[1]
}
