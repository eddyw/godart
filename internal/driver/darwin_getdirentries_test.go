package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDarwinGetdirentries(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Getdirentries requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/getdirentries\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"os";"path/filepath";"sort";"syscall")
func main(){
 base:=uintptr(73)
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  data:=[]byte{1,2,3};n,err:=syscall.Getdirentries(-1,data,&base)
  if n!=0||!errors.Is(err,errors.ErrUnsupported)||base!=73||!bytes.Equal(data,[]byte{1,2,3}){panic("unsupported contract")}
  fmt.Println("getdirentries unsupported");return
 }
 data:=[]byte{1,2,3};n,err:=syscall.Getdirentries(-1,data,&base)
 fmt.Println("invalid",n,err,base,data)
 dir,err:=os.MkdirTemp("","getdirentries-");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 for _,name:=range []string{"alpha","beta","long-file-name","日本語"}{if err:=os.WriteFile(filepath.Join(dir,name),nil,0600);err!=nil{panic(err)}}
 fd,err:=syscall.Open(dir,syscall.O_RDONLY,0);if err!=nil{panic(err)};defer syscall.Close(fd)
 for _,size:=range []int{0,1,16}{
  data:=bytes.Repeat([]byte{91},size);n,err:=syscall.Getdirentries(fd,data,&base)
  pos,e:=syscall.Seek(fd,0,1);fmt.Println("small",size,n,err,base,bytes.Equal(data,bytes.Repeat([]byte{91},size)),pos,e)
 }
 for _,size:=range []int{64,2048}{
  if _,err:=syscall.Seek(fd,0,0);err!=nil{panic(err)}
  var names []string
  for calls:=0;;calls++{
   if calls>20{panic("directory cursor stalled")}
   data:=bytes.Repeat([]byte{91},size+4);n,err:=syscall.Getdirentries(fd,data[2:2+size],&base)
   if err!=nil{panic(err)}
   if data[0]!=91||data[1]!=91||!bytes.Equal(data[2+n:],bytes.Repeat([]byte{91},size+2-n)){panic("caller padding changed")}
   if base!=73{panic("base changed")}
   if n==0{break}
   _,_,names=syscall.ParseDirent(data[2:2+n],-1,names)
  }
  sort.Strings(names);pos,err:=syscall.Seek(fd,0,1);fmt.Println("names",size,names,pos,err)
 }
 if _,err:=syscall.Seek(fd,0,0);err!=nil{panic(err)}
 n,err=syscall.Getdirentries(fd,make([]byte,2048),nil);fmt.Println("nil base",n>0,err)
 file,err:=syscall.Open(filepath.Join(dir,"alpha"),syscall.O_RDONLY,0);if err!=nil{panic(err)}
 data=[]byte{1,2,3};n,err=syscall.Getdirentries(file,data,&base);fmt.Println("file",n,err,base,data)
 syscall.Close(file);n,err=syscall.Getdirentries(file,data,&base);fmt.Println("closed",n,err,base,data)
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
			if got.Exit != 0 || got.Text != "getdirentries unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

}
