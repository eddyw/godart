package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUserDirectories(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("this adapter covers macOS and Unix user directories")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/userdirs\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os")
func main(){
 for _,home:=range []string{"","/user/home","relative-home"}{
  for _,xdg:=range []string{"","/user/xdg","relative-xdg",".","/root/../xdg/"}{
   for _,entry:=range [][2]string{{"HOME",home},{"XDG_CONFIG_HOME",xdg},{"XDG_CACHE_HOME",xdg}}{
    if err:=os.Setenv(entry[0],entry[1]);err!=nil{panic(err)}
   }
   config,ce:=os.UserConfigDir();cache,ca:=os.UserCacheDir()
   fmt.Printf("home=%q xdg=%q config=%q error=%v cache=%q error=%v\n",home,xdg,config,ce,cache,ca)
  }
 }
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot"}
	if runtime.GOOS == "linux" {
		targets = append(targets, "js", "wasm")
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}

func TestUserDirectoriesUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("this adapter covers macOS and Unix user directories")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/userdirs\n\ngo 1.27\n")
	names := []string{"TestUserCacheDir", "TestUserConfigDir", "TestUserHomeDir"}
	if runtime.GOOS == "linux" {
		names = append(names, "TestUserCacheDirXDGConfigDirEnvVar", "TestUserConfigDirXDGConfigDirEnvVar")
	}
	pattern := "^(" + strings.Join(names, "|") + ")$"
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/os/os_test.go"), names)
	writeFile(t, filepath.Join(dir, "userdirs_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot"}
	if runtime.GOOS == "linux" {
		targets = append(targets, "js", "wasm")
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			targetPattern := pattern
			if target == "js" || target == "wasm" {
				// The other original tests require Stat, which web targets do not support.
				targetPattern = "^TestUser(Cache|Config)DirXDGConfigDirEnvVar$"
			}
			args := upstreamTestArgs("-test.run=" + targetPattern)
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "user-directories", "os", targetPattern, target, filepath.Join(runtime.GOROOT(), "src/os"), got.Text, got.Exit, true)
		})
	}
}
