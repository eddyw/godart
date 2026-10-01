package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDarwinMkfifo(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Mkfifo requires macOS")
	}
	testMkfifo(t)
}

func TestLinuxMkfifo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Mkfifo requires Linux")
	}
	testMkfifo(t)
}

func testMkfifo(t *testing.T) {
	t.Helper()
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/mkfifo\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func main(){
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  if !errors.Is(syscall.Mkfifo("missing/fifo",0600),errors.ErrUnsupported){panic("FIFO creation must be unsupported")}
  fmt.Println("mkfifo unsupported");return
 }
 fmt.Println("first NUL",syscall.Mkfifo("a\x00b",0600))
 dir,e:=os.MkdirTemp("","mkfifo-");if e!=nil{panic(e)};defer os.RemoveAll(dir)
 fmt.Println("first missing",syscall.Mkfifo(filepath.Join(dir,"missing","first"),0600))
 old:=syscall.Umask(0027);defer syscall.Umask(old)
 for i,mode:=range []uint32{0600,0777,04777,01777,010600,0100600,020600,0xffff0600}{
  path:=filepath.Join(dir,fmt.Sprint(i));err:=syscall.Mkfifo(path,mode);fmt.Println("create",i,err)
  if err==nil{var st syscall.Stat_t;if e:=syscall.Stat(path,&st);e!=nil{panic(e)};fmt.Printf("mode %d %o %v\n",i,st.Mode,st.Mode&syscall.S_IFMT==syscall.S_IFIFO);fmt.Println("duplicate",syscall.Mkfifo(path,0600))}
 }
 file:=filepath.Join(dir,"file");if e:=os.WriteFile(file,nil,0600);e!=nil{panic(e)}
 link:=filepath.Join(dir,"link");if e:=os.Symlink("file",link);e!=nil{panic(e)}
 dangling:=filepath.Join(dir,"dangling");if e:=os.Symlink("absent",dangling);e!=nil{panic(e)}
 for i,path:=range []string{file,link,dangling,filepath.Join(file,"child"),filepath.Join(dir,"absent","fifo"),"", "a\x00b"}{fmt.Println("bad path",i,syscall.Mkfifo(path,0600))}
 raw:=filepath.Join(dir,string([]byte{0xff}));fmt.Println("byte path",syscall.Mkfifo(raw,0600));var st syscall.Stat_t;fmt.Println("byte stat",syscall.Stat(raw,&st),st.Mode&syscall.S_IFMT==syscall.S_IFIFO)
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
			if got.Exit != 0 || got.Text != "mkfifo unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

}
