package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxNetlinkMessageContracts(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Netlink message contracts require Linux platform types")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/netlinkmessages\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("encoding/binary";"encoding/json";"fmt";"syscall")
type result struct { Messages []syscall.NetlinkMessage; Capacities []int; Errno int }
func header(length uint32) []byte {
 b := make([]byte, syscall.NLMSG_HDRLEN)
 binary.NativeEndian.PutUint32(b[0:4], length)
 binary.NativeEndian.PutUint16(b[4:6], 0x8021)
 binary.NativeEndian.PutUint16(b[6:8], 0xf234)
 binary.NativeEndian.PutUint32(b[8:12], 0xfedcba98)
 binary.NativeEndian.PutUint32(b[12:16], 0x87654321)
 return b
}
func observe(wire []byte) result {
 messages, err := syscall.ParseNetlinkMessage(wire)
 r := result{Messages: messages}
 if err != nil { r.Errno = int(err.(syscall.Errno)) }
 for _, message := range messages { r.Capacities = append(r.Capacities, cap(message.Data)) }
 return r
}
func main() {
 valid := append(header(19), 0, 255, 128, 77)
 two := append(append([]byte{}, valid...), header(16)...)
 cases := []struct{name string; data []byte; capacity int}{
  {"nil", nil, 0},
  {"empty", []byte{}, 0},
  {"short", make([]byte,15), 15},
  {"zero-length", header(0), 16},
  {"short-length", header(15), 16},
  {"oversized-length", header(20), 16},
  {"maximum-length", header(^uint32(0)), 16},
  {"missing-padding", append(header(17),1), 17},
  {"empty-payload", header(16), 16},
  {"one-byte-payload", append(header(17),1,2,3,4), 20},
  {"binary-payload", valid, 20},
  {"spare-capacity", valid, 64},
  {"two-messages", two, 36},
  {"trailing-bytes", append(append([]byte{},valid...),make([]byte,15)...), 35},
  {"invalid-after-valid", append(append([]byte{},valid...),header(15)...), 36},
 }
 results := make(map[string]result)
 for _, test := range cases {
  var wire []byte
  if test.data != nil { wire = make([]byte,len(test.data),test.capacity); copy(wire,test.data) }
  results[test.name] = observe(wire)
 }
 wire := make([]byte,20,64)
 copy(wire,valid)
 alias := observe(wire)
 wire[16] = 254
 alias.Messages[0].Data[1] = 253
 binary.NativeEndian.PutUint32(wire[0:4],1)
 results["data-alias-header-copy"] = alias
 window := make([]byte,22,66)
 copy(window[2:],valid)
 results["offset-input"] = observe(window[2:22])
 data, _ := json.Marshal(struct { Results map[string]result; WireBytes []byte }{results,wire[16:19]})
 fmt.Println(string(data))
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
