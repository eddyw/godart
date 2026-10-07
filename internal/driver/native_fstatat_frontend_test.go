package driver_test

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeFstatatFrontendExecution(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native fstatat acceptance requires macOS or Linux")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nativefstatatfrontend\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"io/fs";"os";"path/filepath";"syscall")
func main(){
 dir,err:=os.MkdirTemp("","rootstaterrno");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 if err:=os.WriteFile(filepath.Join(dir,"file"),[]byte("abc"),0600);err!=nil{panic(err)}
 if err:=os.Symlink("file",filepath.Join(dir,"link"));err!=nil{panic(err)}
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 missing,invalid,regular,link:=0,0,0,0
 for i:=0;i<32;i++{
  _,err:=root.Lstat("missing");if errors.Is(err,fs.ErrNotExist){missing++}
  _,err=root.Lstat("bad\x00name");if errors.Is(err,syscall.EINVAL){invalid++}
  info,err:=root.Lstat("file");if err==nil&&info.Size()==3&&info.Mode().IsRegular(){regular++}
  info,err=root.Lstat("link");if err==nil&&info.Mode()&fs.ModeSymlink!=0{link++}
 }
 fmt.Println("stat calls",missing,invalid,regular,link)
}
`)
	const expected = "stat calls 32 32 32 32\n"
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 || strings.TrimSpace(want.Text) != strings.TrimSpace(expected) {
		t.Fatalf("native fstatat contract: %+v", want)
	}
	for _, frontend := range []string{"source", "resident"} {
		t.Run(frontend, func(t *testing.T) {
			opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t), Frontend: frontend}
			var output, stderr bytes.Buffer
			code, err := driver.Run(opt, nil, &output, &stderr)
			if err != nil || code != 0 || output.String() != expected || stderr.Len() != 0 {
				t.Fatalf("code %d, error %v, stdout %q, stderr %q", code, err, output.String(), stderr.String())
			}
		})
	}
}
