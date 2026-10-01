package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxXattr(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("extended attributes require Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/xattr\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"os";"path/filepath";"sort";"strings";"syscall")
func main(){
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  b:=[]byte{7,8};n,e:=syscall.Getxattr("missing","user.test",b);if n!=0||!errors.Is(e,errors.ErrUnsupported)||!bytes.Equal(b,[]byte{7,8}){panic("get unsupported")}
  n,e=syscall.Listxattr("missing",b);if n!=0||!errors.Is(e,errors.ErrUnsupported)||!bytes.Equal(b,[]byte{7,8}){panic("list unsupported")}
  if !errors.Is(syscall.Setxattr("missing","user.test",b,0),errors.ErrUnsupported){panic("set unsupported")}
  if !errors.Is(syscall.Removexattr("missing","user.test"),errors.ErrUnsupported){panic("remove unsupported")}
  fmt.Println("xattr unsupported");return
 }
 dir,e:=os.MkdirTemp("","xattr-");if e!=nil{panic(e)};defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"file");if e=os.WriteFile(path,nil,0600);e!=nil{panic(e)}
 key:="user.godart";value:=[]byte{0,1,127,128,255}
 n,e:=syscall.Getxattr(path,key,nil);fmt.Println("absent",n,e)
 fmt.Println("replace absent",syscall.Setxattr(path,key,value,2))
 if e=syscall.Setxattr(path,key,value,1);e!=nil{panic(e)}
 fmt.Println("duplicate",syscall.Setxattr(path,key,value,1))
 for _,size:=range []int{0,1,4,5,8}{
  b:=bytes.Repeat([]byte{99},size);n,e:=syscall.Getxattr(path,key,b);fmt.Println("get",size,n,e,b)
 }
 for _,size:=range []int{0,1,len(key),len(key)+1,len(key)+4}{
  b:=bytes.Repeat([]byte{99},size);n,e:=syscall.Listxattr(path,b);fmt.Println("list",size,n,e,b)
 }
 fmt.Println("replace",syscall.Setxattr(path,key,[]byte{9},2))
 fmt.Println("empty",syscall.Setxattr(path,key,nil,0));n,e=syscall.Getxattr(path,key,nil);fmt.Println("empty size",n,e)
 fmt.Println("bad flags",syscall.Setxattr(path,key,value,4))
 link:=filepath.Join(dir,"link");if e=os.Symlink("file",link);e!=nil{panic(e)}
 fmt.Println("link set",syscall.Setxattr(link,key,value,0));b:=make([]byte,5);n,e=syscall.Getxattr(path,key,b);fmt.Println("link value",n,e,b)
 raw:=filepath.Join(dir,string([]byte{0xff}));if e=os.WriteFile(raw,nil,0600);e!=nil{panic(e)}
 rawKey:="user."+string([]byte{0xfe});fmt.Println("raw set",syscall.Setxattr(raw,rawKey,value,0));n,e=syscall.Getxattr(raw,rawKey,b);fmt.Println("raw get",n,e,b)
 names:=make([]byte,256);n,e=syscall.Listxattr(raw,names);if e!=nil{panic(e)};keys:=strings.Split(string(names[:n]),"\x00");sort.Strings(keys);fmt.Printf("raw names %q\n",keys)
 span:=[]byte{90,1,2,3,91};fmt.Println("slice set",syscall.Setxattr(path,key,span[1:4],0));span=[]byte{90,8,8,8,91};n,e=syscall.Getxattr(path,key,span[1:4]);fmt.Println("slice get",n,e,span)
 fmt.Println("remove",syscall.Removexattr(path,key));fmt.Println("remove absent",syscall.Removexattr(path,key))
 for i,item:=range [][2]string{{"a\x00b",key},{path,"user.a\x00b"},{filepath.Join(dir,"missing"),key},{path,""},{path,"user."}}{
  b:=[]byte{6,7};n,e:=syscall.Getxattr(item[0],item[1],b);fmt.Println("bad get",i,n,e,b)
  fmt.Println("bad set",i,syscall.Setxattr(item[0],item[1],value,0));fmt.Println("bad remove",i,syscall.Removexattr(item[0],item[1]))
 }
 for _,path:=range []string{"a\x00b",filepath.Join(dir,"missing")} {b:=[]byte{6,7};n,e:=syscall.Listxattr(path,b);fmt.Println("bad list",n,e,b)}
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
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unsupported"})
			if got.Exit != 0 || got.Text != "xattr unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}
}
