package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxProcessTimes(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Kernel process observations require Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"strconv";"strings";"syscall")
func must(err error){if err!=nil{panic(err)}}
func counters()[4]int64{data,err:=os.ReadFile("/proc/self/stat");must(err);end:=strings.LastIndexByte(string(data),')');if end<0{panic("stat comm")};fields:=strings.Fields(string(data[end+1:]));if len(fields)<15{panic("stat fields")};var result [4]int64;for i:=range result{result[i],err=strconv.ParseInt(fields[11+i],10,64);must(err)};return result}
func main(){
 tid:=syscall.Gettid();data,err:=os.ReadFile(fmt.Sprintf("/proc/self/task/%d/status",tid));must(err);ids:=make(map[string]int)
 for _,line:=range strings.Split(string(data),"\n"){fields:=strings.Fields(line);if len(fields)==2&&(fields[0]=="Pid:"||fields[0]=="Tgid:"||fields[0]=="PPid:"){value,err:=strconv.Atoi(fields[1]);must(err);ids[fields[0]]=value}}
 fmt.Println("thread",tid>0,ids["Pid:"]==tid,ids["Tgid:"]==syscall.Getpid(),ids["PPid:"]==syscall.Getppid())
 before:=counters();value:=syscall.Tms{Utime:-7,Stime:-11,Cutime:-13,Cstime:-17};ticks,err:=syscall.Times(&value);must(err);after:=counters();observed:=[4]int64{value.Utime,value.Stime,value.Cutime,value.Cstime}
 for i,n:=range observed{fmt.Println("counter",i,n>=0,n>=before[i],n<=after[i])}
 saved:=value;nilTicks,err:=syscall.Times(nil);must(err);later:=syscall.Tms{};lastTicks,err:=syscall.Times(&later);must(err)
 fmt.Println("ticks",ticks>0,nilTicks>=ticks,lastTicks>=nilTicks,value==saved)
 fmt.Println("later",later.Utime>=value.Utime,later.Stime>=value.Stime,later.Cutime>=value.Cutime,later.Cstime>=value.Cstime)
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
