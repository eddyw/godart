package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxSysinfo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Sysinfo requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"strconv";"strings";"syscall")
func must(err error){if err!=nil{panic(err)}}
func uptime()int64{data,err:=os.ReadFile("/proc/uptime");must(err);fields:=strings.Fields(string(data));if len(fields)!=2{panic("uptime fields")};seconds,err:=strconv.ParseInt(strings.Split(fields[0],".")[0],10,64);must(err);return seconds}
func memory()map[string]uint64{data,err:=os.ReadFile("/proc/meminfo");must(err);result:=make(map[string]uint64);for _,line:=range strings.Split(string(data),"\n"){fields:=strings.Fields(line);if len(fields)>0&&(fields[0]=="MemTotal:"||fields[0]=="SwapTotal:"){if len(fields)!=3||fields[2]!="kB"{panic("memory fields")};value,err:=strconv.ParseUint(fields[1],10,64);must(err);result[fields[0]]=value*1024}};if _,ok:=result["MemTotal:"];!ok{panic("memory total")};if _,ok:=result["SwapTotal:"];!ok{panic("swap total")};return result}
func main(){
 fmt.Println("nil",syscall.Sysinfo(nil)==syscall.EFAULT)
 box:=struct{Before uint64;Value syscall.Sysinfo_t;After uint64}{Before:17,After:19}
 box.Value=syscall.Sysinfo_t{Uptime:-1,Loads:[3]uint64{31,37,41},Totalram:43,Freeram:47,Sharedram:53,Bufferram:59,Totalswap:61,Freeswap:67,Procs:71,Pad:73,Pad_cgo_0:[4]byte{2,3,5,7},Totalhigh:79,Freehigh:83,Unit:89,Pad_cgo_1:[4]byte{11,13,17,19}}
 before:=uptime();must(syscall.Sysinfo(&box.Value));after:=uptime();actual:=memory();info:=box.Value
 fmt.Println("uptime",info.Uptime>=before,info.Uptime<=after+1)
 fmt.Println("memory",info.Unit>0,info.Totalram*uint64(info.Unit)==actual["MemTotal:"],info.Totalswap*uint64(info.Unit)==actual["SwapTotal:"],info.Freeram<=info.Totalram,info.Freeswap<=info.Totalswap,info.Freehigh<=info.Totalhigh)
 fmt.Println("processes",info.Procs>0)
 fmt.Println("padding",info.Pad,info.Pad_cgo_0,info.Pad_cgo_1)
 saved:=box.Value;var other syscall.Sysinfo_t;must(syscall.Sysinfo(&other));other.Loads[0]=0
 fmt.Println("ownership",box.Before==17,box.After==19,box.Value==saved)
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
