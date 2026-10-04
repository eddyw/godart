package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxNetlinkRouteAttributes(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Netlink route attributes require Linux platform types")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/netlinkattributes\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "encoding/binary"
 "encoding/json"
 "fmt"
 "syscall"
)

type result struct {
 Attributes []syscall.NetlinkRouteAttr
 Capacities []int
 Errno int
 Panic bool
}

func observe(message *syscall.NetlinkMessage) (r result) {
 defer func() { if recover() != nil { r.Panic = true } }()
 attributes, err := syscall.ParseNetlinkRouteAttr(message)
 r.Attributes = attributes
 if err != nil { r.Errno = int(err.(syscall.Errno)) }
 for _, attribute := range attributes { r.Capacities = append(r.Capacities, cap(attribute.Value)) }
 return r
}

func prefix(kind uint16) int {
 switch kind {
 case syscall.RTM_NEWLINK, syscall.RTM_DELLINK: return syscall.SizeofIfInfomsg
 case syscall.RTM_NEWADDR, syscall.RTM_DELADDR: return syscall.SizeofIfAddrmsg
 case syscall.RTM_NEWROUTE, syscall.RTM_DELROUTE: return syscall.SizeofRtMsg
 }
 return 0
}

func header(length uint16) []byte {
 b := make([]byte,syscall.SizeofRtAttr)
 binary.NativeEndian.PutUint16(b,length)
 binary.NativeEndian.PutUint16(b[2:],0xf234)
 return b
}

func message(kind uint16, payload []byte, capacity int) *syscall.NetlinkMessage {
 n := prefix(kind)
 data := make([]byte,n+len(payload),n+capacity)
 copy(data[n:],payload)
 return &syscall.NetlinkMessage{Header:syscall.NlMsghdr{Type:kind},Data:data}
}

func main() {
 results := make(map[string]result)
 valid := append(header(7),0,255,128,77)
 for _, kind := range []uint16{syscall.RTM_NEWLINK,syscall.RTM_DELLINK,syscall.RTM_NEWADDR,syscall.RTM_DELADDR,syscall.RTM_NEWROUTE,syscall.RTM_DELROUTE} {
  results[fmt.Sprint("recognized-",kind)] = observe(message(kind,valid,8))
  results[fmt.Sprint("empty-",kind)] = observe(message(kind,nil,0))
  results[fmt.Sprint("short-prefix-",kind)] = observe(&syscall.NetlinkMessage{Header:syscall.NlMsghdr{Type:kind},Data:make([]byte,prefix(kind)-1)})
 }
 results["nil-message"] = observe(nil)
 results["unsupported-zero"] = observe(&syscall.NetlinkMessage{})
 results["unsupported-high"] = observe(&syscall.NetlinkMessage{Header:syscall.NlMsghdr{Type:^uint16(0)}})
 kind := uint16(syscall.RTM_NEWLINK)
 for _, test := range []struct{name string; payload []byte; capacity int}{
  {"short-header",make([]byte,3),3},
  {"zero-length",header(0),4},
  {"short-length",header(3),4},
  {"oversized-length",header(5),4},
  {"maximum-length",header(^uint16(0)),4},
  {"missing-alignment",append(header(7),1,2,3),7},
  {"missing-alignment-spare-capacity",append(header(7),1,2,3),64},
  {"empty-value",header(4),4},
  {"one-byte-value",append(header(5),1,2,3,4),8},
  {"binary-value",valid,8},
  {"spare-capacity",valid,64},
  {"two-attributes",append(append([]byte{},valid...),header(4)...),12},
  {"trailing-bytes",append(append([]byte{},valid...),1,2,3),11},
  {"invalid-after-valid",append(append([]byte{},valid...),header(3)...),12},
 } {
  results[test.name] = observe(message(kind,test.payload,test.capacity))
 }
 m := message(kind,valid,64)
 alias := observe(m)
 offset := prefix(kind)
 m.Data[offset+4] = 254
 alias.Attributes[0].Value[1] = 253
 binary.NativeEndian.PutUint16(m.Data[offset:],1)
 binary.NativeEndian.PutUint16(m.Data[offset+2:],0)
 results["value-alias-header-copy"] = alias
 window := make([]byte,offset+10,offset+66)
 copy(window[offset+2:],valid)
 results["offset-input"] = observe(&syscall.NetlinkMessage{Header:syscall.NlMsghdr{Type:kind},Data:window[2:offset+10]})
 data,_ := json.Marshal(struct {Results map[string]result; WireBytes []byte}{results,m.Data[offset+4:offset+7]})
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
