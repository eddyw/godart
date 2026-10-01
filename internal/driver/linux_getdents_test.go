package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxGetdents(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Getdents requires Linux")
	}
	programs := []struct{ name, source string }{
		{"first", `package main
import "syscall"
func main(){
 n,err:=syscall.ReadDirent(-1,nil);println(n,err==syscall.EBADF)
 n,err=syscall.ReadDirent(-1,[]byte{17,23});println(n,err==syscall.EBADF)
}
`},
		{"directory", `package main
import("bytes";"encoding/binary";"fmt";"os";"path/filepath";"slices";"syscall";"unsafe")
func must(err error){if err!=nil{panic(err)}}
func show(label string,n int,err error){
 code:=syscall.Errno(0);if err!=nil{code=err.(syscall.Errno)}
 fmt.Println(label,n,uint64(code))
}
func main(){
 base,err:=os.MkdirTemp("","getdents-");must(err);defer os.RemoveAll(base)
 for _,name:=range []string{"short","longer-entry-name","r\xffw","r\ufffdw"}{must(os.WriteFile(filepath.Join(base,name),[]byte("value"),0600))}
 must(os.Mkdir(filepath.Join(base,"subdir"),0700));must(os.Symlink("short",filepath.Join(base,"symlink")))
 regular,err:=syscall.Open(filepath.Join(base,"short"),syscall.O_RDONLY,0);must(err)
 n,err:=syscall.ReadDirent(regular,make([]byte,256));show("regular",n,err);must(syscall.Close(regular))
 fd,err:=syscall.Open(base,syscall.O_RDONLY|syscall.O_DIRECTORY,0);must(err);defer syscall.Close(fd)
 for _,size:=range []int{0,1}{n,err:=syscall.ReadDirent(fd,make([]byte,size));show("small",n,err)}
 duplicate,err:=syscall.Dup(fd);must(err)
 moved:=base+"-moved";must(os.Rename(base,moved));defer os.RemoveAll(moved)
 const lengthOffset=int(unsafe.Offsetof(syscall.Dirent{}.Reclen))
 const nameOffset=int(unsafe.Offsetof(syscall.Dirent{}.Name))
 const typeOffset=int(unsafe.Offsetof(syscall.Dirent{}.Type))
 var names,records []string
 total:=0;padding,guards:=true,true
 for turn:=0;;turn++{
  marker:=byte(0xa5);if turn%2!=0{marker=0x5a}
  data:=bytes.Repeat([]byte{marker},144)
  current:=fd;if turn%2!=0{current=duplicate}
  n,err:=syscall.ReadDirent(current,data[8:136]);must(err)
  for i,b:=range data{if (i<8||i>=8+n)&&b!=marker{guards=false}}
  if n==0{break}
  total+=n
  consumed,count,parsed:=syscall.ParseDirent(data[8:8+n],-1,nil)
  if consumed!=n||count!=len(parsed){panic("parsed record count")}
  names=append(names,parsed...)
  for cursor:=8;cursor<8+n;{
   length:=int(binary.LittleEndian.Uint16(data[cursor+lengthOffset:]))
   if length<=nameOffset||cursor+length>8+n{panic("record length")}
   record:=data[cursor:cursor+length]
   zero:=bytes.IndexByte(record[nameOffset:],0);if zero<0{panic("unterminated name")};zero+=nameOffset
   records=append(records,fmt.Sprintf("%x/%d",record[nameOffset:zero],record[typeOffset]))
   for _,b:=range record[zero+1:]{if b!=marker{padding=false}}
   cursor+=length
  }
 }
 slices.Sort(names);slices.Sort(records)
 for _,name:=range names{fmt.Printf("name %x\n",name)}
 fmt.Println("records",records)
 fmt.Println("buffer",padding,guards,total)
 position,err:=syscall.Seek(fd,0,0);must(err);fmt.Println("rewind",position)
 data:=make([]byte,1024);n,err=syscall.ReadDirent(duplicate,data);must(err)
 _,_,again:=syscall.ParseDirent(data[:n],-1,nil);slices.Sort(again)
 fmt.Println("shared offset",slices.Equal(names,again),n==total)
 must(syscall.Close(duplicate));n,err=syscall.ReadDirent(duplicate,data);show("closed",n,err)
}
`},
	}
	for _, api := range []string{"ReadDirent", "Getdents"} {
		for _, program := range programs {
			t.Run(api+"/"+program.name, func(t *testing.T) {
				dir, out := t.TempDir(), t.TempDir()
				writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/getdents\n\ngo 1.27\n")
				writeFile(t, filepath.Join(dir, "main.go"), strings.ReplaceAll(program.source, "syscall.ReadDirent", "syscall."+api))
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
			})
		}
	}
}
