package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTemporaryRandomNames(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("temporary file creation requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/temporarynames\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"strconv";"strings")
func must(err error){if err!=nil{panic(err)}}
func valid(name,prefix,suffix string)bool{
 name=filepath.Base(name)
 if !strings.HasPrefix(name,prefix)||!strings.HasSuffix(name,suffix){return false}
 digits:=name[len(prefix):len(name)-len(suffix)]
 n,err:=strconv.ParseUint(digits,10,32)
 return err==nil&&strconv.FormatUint(n,10)==digits
}
func main(){
 root,err:=os.MkdirTemp(".","temporary-names");must(err);defer os.RemoveAll(root)
 for _,pattern:=range []string{"","file","pre*suf","pre*mid*suf"}{
  prefix,suffix:=pattern,"";if i:=strings.LastIndexByte(pattern,'*');i>=0{prefix,suffix=pattern[:i],pattern[i+1:]}
  fileOK,dirOK,exclusive:=true,true,true
  for i:=0;i<16;i++{
   f,err:=os.CreateTemp(root,pattern);must(err)
   fileOK=fileOK&&valid(f.Name(),prefix,suffix)
   must(f.Close())
   other,e:=os.OpenFile(f.Name(),os.O_CREATE|os.O_EXCL,0600);if other!=nil{other.Close()};exclusive=exclusive&&os.IsExist(e)
   name,err:=os.MkdirTemp(root,pattern);must(err)
   dirOK=dirOK&&valid(name,prefix,suffix)
   exclusive=exclusive&&os.IsExist(os.Mkdir(name,0700))
  }
  fmt.Printf("pattern=%q file=%t directory=%t exclusive=%t\n",pattern,fileOK,dirOK,exclusive)
 }
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
}
