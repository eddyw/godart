package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxGetcwd(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Getcwd requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/getcwd\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func probe(label,want string){
 for index,size:=range []int{0,1,len(want),len(want)+1,len(want)+17}{
  data:=bytes.Repeat([]byte{0xa5},size+16)
  n,err:=syscall.Getcwd(data[8:8+size]);code:=syscall.Errno(0)
  if err!=nil{code=err.(syscall.Errno)}
  count:=n==-1;content:=true;end:=8
  if err==nil{count=n==len(want)+1;content=bytes.Equal(data[8:8+n],append([]byte(want),0));end+=n}
  guards:=bytes.Equal(data[:8],bytes.Repeat([]byte{0xa5},8))&&bytes.Equal(data[end:],bytes.Repeat([]byte{0xa5},len(data)-end))
  fmt.Println(label,index,count,content,guards,uint64(code))
 }
}
func main(){
 n,err:=syscall.Getcwd(nil);fmt.Println("nil",n,err==syscall.ERANGE)
 old,err:=os.Open(".");must(err);defer old.Close();defer old.Chdir()
 base,err:=os.MkdirTemp("","getcwd-");must(err);defer os.RemoveAll(base)
 must(os.Chdir(base));base,err=syscall.Getwd();must(err)
 for _,name:=range []string{"ascii","unicode-世界","raw-\xff"}{
  path:=filepath.Join(base,name);must(os.Mkdir(path,0700));must(os.Chdir(path));probe(name,path)
  must(os.Chdir(base))
 }
 path:=filepath.Join(base,"renamed");must(os.Mkdir(path,0700));must(os.Chdir(path))
 moved:=filepath.Join(base,"moved");must(os.Rename(path,moved));probe("renamed",moved)
 must(os.Remove(moved))
 data:=bytes.Repeat([]byte{0x5a},4096);n,err=syscall.Getcwd(data)
 fmt.Println("removed",n,err==syscall.ENOENT,bytes.Equal(data,bytes.Repeat([]byte{0x5a},4096)))
 must(old.Chdir())
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
	for _, target := range []string{"vm", "aot"} {
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
