package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestInterfaceMetadata(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native interface metadata requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/interfacemetadata\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("encoding/json"; "net"; "os"; "slices")
func main() {
 interfaces, err := net.Interfaces()
 if err != nil { panic(err) }
 slices.SortFunc(interfaces, func(a,b net.Interface) int { return a.Index-b.Index })
 if err := json.NewEncoder(os.Stdout).Encode(interfaces); err != nil { panic(err) }
}
`)
	native := filepath.Join(dir, "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := runProgram(t, dir, native)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, out, "dart", "bin/main.dart")
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("interface metadata differs from native Go: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}

func TestInterfaceAddresses(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native interface metadata requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/interfaceaddresses\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/interface_test.go"), []string{"routeStats", "validateInterfaceUnicastAddrs"})
	source = strings.Replace(source, "package net\n", "package main\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream.go"), source)
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("encoding/json"; "net"; "os"; "sort")
func main() {
 all, err := net.InterfaceAddrs()
 if err != nil { panic(err) }
 if _, err := validateInterfaceUnicastAddrs(all); err != nil { panic(err) }
 interfaces, err := net.Interfaces()
 if err != nil { panic(err) }
 sort.Slice(interfaces, func(i,j int) bool { return interfaces[i].Index < interfaces[j].Index })
 type record struct { Index int; IP []byte; Mask []byte }
 records := []record{}
 for _, iface := range interfaces {
  addresses, err := iface.Addrs()
  if err != nil { panic(err) }
  if _, err := validateInterfaceUnicastAddrs(addresses); err != nil { panic(err) }
  sort.Slice(addresses, func(i,j int) bool { return addresses[i].String() < addresses[j].String() })
  for _, address := range addresses {
   ip := address.(*net.IPNet)
   records = append(records, record{iface.Index, ip.IP, ip.Mask})
  }
 }
 if err := json.NewEncoder(os.Stdout).Encode(records); err != nil { panic(err) }
}
`)
	native := filepath.Join(dir, "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := runProgram(t, dir, native)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, out, "dart", "bin/main.dart")
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("interface addresses differ from native Go: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}
