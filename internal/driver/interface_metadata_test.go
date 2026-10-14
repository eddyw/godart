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
 interfaces = append(interfaces, net.Interface{})
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
			// Interface addresses can change while a target compiles.
			want := runProgram(t, dir, native)
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("interface addresses differ from native Go: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}

func TestInterfaceMulticastAddresses(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native interface metadata requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/interfacemulticastaddresses\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/interface_test.go"), []string{"routeStats", "validateInterfaceMulticastAddrs"})
	source = strings.Replace(source, "package net\n", "package main\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream.go"), source)
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("encoding/json"; "net"; "os"; "sort")
func membership(iface *net.Interface, group net.IP) *net.IPAddr {
 addresses, err := iface.MulticastAddrs()
 if err != nil { panic(err) }
 if _, err := validateInterfaceMulticastAddrs(addresses); err != nil { panic(err) }
 for _, address := range addresses {
  ip := address.(*net.IPAddr)
  if ip.IP.Equal(group) { return ip }
 }
 return nil
}
func main() {
 interfaces, err := net.Interfaces()
 if err != nil { panic(err) }
 sort.Slice(interfaces, func(i,j int) bool { return interfaces[i].Index < interfaces[j].Index })
 var selected *net.Interface
 for i := range interfaces {
  iface := &interfaces[i]
  if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 || iface.Flags&net.FlagLoopback != 0 { continue }
  addresses, err := iface.Addrs()
  if err != nil { panic(err) }
  for _, address := range addresses {
   if address.(*net.IPNet).IP.To4() != nil { selected = iface; break }
  }
  if selected != nil { break }
 }
 if selected == nil { panic("no multicast interface with an IPv4 address") }
 type record struct { Case int; Index int; IP []byte; Zone string }
 results := []record{}
 for i, iface := range []*net.Interface{selected, {Index: selected.Index}, {Index: selected.Index, Name: "absent-multicast-interface"}, nil, {}} {
  group := net.IPv4(239, 255, 77, byte(77+i))
  conn, err := net.ListenMulticastUDP("udp4", iface, &net.UDPAddr{IP: group})
  if err != nil { panic(err) }
  found := false
  for j := range interfaces {
   joined := membership(&interfaces[j], group)
   if joined != nil {
    results = append(results, record{i, interfaces[j].Index, joined.IP, joined.Zone})
    found = true
   }
  }
  if !found { panic("joined multicast group is absent") }
  if err := conn.Close(); err != nil { panic(err) }
 }
 if err := json.NewEncoder(os.Stdout).Encode(results); err != nil { panic(err) }
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
				t.Fatalf("multicast interface addresses differ from native Go: exit %d\nGo: %s\nGart: %s", got.Exit, want, got.Text)
			}
		})
	}
}
