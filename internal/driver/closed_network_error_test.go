package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestClosedNetworkError(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/closederror\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("encoding/json";"errors";"fmt";"net";"os")
func main(){
 var n net.Error
 ok:=errors.As(net.ErrClosed,&n)
 r:=struct{Type,Text string;NetworkError,Timeout,Temporary,Wrapped bool}{Type:fmt.Sprintf("%T",net.ErrClosed),Text:net.ErrClosed.Error(),NetworkError:ok,Wrapped:errors.Is(fmt.Errorf("wrapped: %w",net.ErrClosed),net.ErrClosed)}
 if ok{r.Timeout=n.Timeout();r.Temporary=n.Temporary()}
 if err:=json.NewEncoder(os.Stdout).Encode(r);err!=nil{panic(err)}
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
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, result.Exit, result.Text)
			}
		})
	}
}
