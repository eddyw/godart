package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/overlay"
	"github.com/eddyw/godart/internal/parity"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	action := flag.String("action", "verify", "inventory, verify, or build")
	pkg := flag.String("pkg", "", "upstream package for build")
	out := flag.String("out", "", "output directory for build")
	manifest := flag.String("manifest", "internal/parity/testdata/"+runtime.GOOS+"-"+runtime.GOARCH+".json", "source manifest")
	goos := flag.String("goos", runtime.GOOS, "inventory operating system")
	goarch := flag.String("goarch", runtime.GOARCH, "inventory architecture")
	flag.Parse()
	if *action == "build" {
		if !slices.Contains(overlay.Surface(), *pkg) || *out == "" {
			return fmt.Errorf("build requires -pkg and -out")
		}
		outDir, err := filepath.Abs(*out)
		if err != nil {
			return err
		}
		runtimeDir, err := filepath.Abs("runtime/goruntime")
		if err != nil {
			return err
		}
		stats, err := driver.BuildWithStats(driver.Options{Dir: filepath.Join(build.Default.GOROOT, "src", filepath.FromSlash(*pkg)), Patterns: []string{"."}, Out: outDir, Runtime: runtimeDir, Tests: true, TrimPath: true})
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(outDir, "test-binaries.json"), data, 0644)
	}
	ctx := build.Default
	ctx.GOOS = *goos
	ctx.GOARCH = *goarch
	tags := exec.Command("go", "list", "-f", `{{join context.ToolTags ","}}`, "runtime")
	tags.Env = append(os.Environ(), "GOOS="+*goos, "GOARCH="+*goarch)
	tagText, err := tags.Output()
	if err != nil {
		return err
	}
	ctx.ToolTags = strings.Split(strings.TrimSpace(string(tagText)), ",")
	inv, err := parity.Discover(ctx)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	switch *action {
	case "inventory":
		return os.WriteFile(*manifest, data, 0644)
	case "verify":
		baseline, err := os.ReadFile(*manifest)
		if err != nil {
			return err
		}
		if !bytes.Equal(baseline, data) {
			return fmt.Errorf("upstream inventory changed: review source, tests, build configuration, and test patches before update")
		}
		fmt.Printf("verified %d upstream packages for %s/%s\n", len(inv.Packages), ctx.GOOS, ctx.GOARCH)
		return nil
	default:
		return fmt.Errorf("unknown action %q", *action)
	}
}
