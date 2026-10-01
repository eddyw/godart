package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestDarwinGetfsstat(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Getfsstat requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/getfsstat\n\ngo 1.27\n")
	program := `package main
import("errors";"fmt";"os";"syscall")
func main(){
 if len(os.Args)>1 && os.Args[1]=="unsupported"{
  buf:=[]syscall.Statfs_t{{Bsize:17,Reserved:[8]uint32{1,2,3}}}
  before:=buf[0]
  n,err:=syscall.Getfsstat(buf,2)
  if n!=-1 || !errors.Is(err,errors.ErrUnsupported) || buf[0]!=before{panic("unsupported call changed output")}
  n,err=syscall.Getfsstat(nil,2)
  if n!=-1 || !errors.Is(err,errors.ErrUnsupported){panic("nil call must be unsupported")}
  fmt.Println("getfsstat unsupported")
  return
 }
 n,err:=syscall.Getfsstat(nil,2)
 if err!=nil || n<=0{panic("native count failed")}
 for _,flags:=range []int{0,3,-1,1<<32|2}{
  count,err:=syscall.Getfsstat(nil,flags)
  fmt.Println("flags",flags,count==n,err)
 }
 buf:=make([]syscall.Statfs_t,n+1)
 marker:=syscall.Statfs_t{Bsize:17,Reserved:[8]uint32{1,2,3}}
 for i:=range buf{buf[i]=marker}
 got,err:=syscall.Getfsstat(buf,2)
 fmt.Println("full",got==n,err,buf[n]==marker)
 for _,stat:=range buf[:got]{
  fmt.Println("space",stat.Blocks>0,stat.Bfree<=stat.Blocks,stat.Bavail<=stat.Bfree)
  stat.Bfree,stat.Bavail,stat.Files,stat.Ffree=0,0,0,0
  fmt.Printf("stat %+v\n",stat)
 }
 one:=make([]syscall.Statfs_t,1)
 got,err=syscall.Getfsstat(one,2)
 fmt.Println("short",got,err,one[0].Fsid==buf[0].Fsid)
 first:=buf[0]
 got,err=syscall.Getfsstat(buf[:0],2)
 fmt.Println("empty",got==n,err,buf[0]==first,buf[n]==marker)
}
`
	writeFile(t, filepath.Join(dir, "main.go"), program)
	upstream := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_bsd_test.go"), []string{"TestGetfsstat", "MNT_NOWAIT"})
	upstream = strings.Replace(upstream, "package syscall_test", "package main", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), upstream)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))

	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot"}
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
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unsupported"})
			if got.Exit != 0 || got.Text != "getfsstat unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

	inventory, err := parity.Discover(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	testOut := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: testOut, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, testOut)
	for _, target := range []string{"vm", "aot"} {
		t.Run("upstream/"+target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=^TestGetfsstat$")
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(testOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, testOut, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "syscall-getfsstat", "syscall", "^TestGetfsstat$", target, dir, result.Text, result.Exit, *inventory, nil, nil, nil, false)
		})
	}

}
