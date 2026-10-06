package driver_test

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestSourceFrontendExecution(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/frontend\n\ngo 1.27.0\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "os")
func main() { fmt.Println(os.Args[1]); println("builtin stderr"); fmt.Fprintln(os.Stderr,"file stderr"); os.Exit(7) }
`)
	opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t)}
	for _, frontend := range []string{"source", "resident"} {
		t.Run(frontend, func(t *testing.T) {
			opt.Frontend = frontend
			var output, stderr bytes.Buffer
			code, err := driver.Run(opt, []string{"two words"}, &output, &stderr)
			if err != nil || code != 7 || output.String() != "two words\n" || stderr.String() != "builtin stderr\nfile stderr\n" {
				t.Fatalf("code %d, error %v, stdout %q, stderr %q", code, err, output.String(), stderr.String())
			}
		})
	}
	writeFile(t, filepath.Join(dir, "main_test.go"), `package main
import "testing"
func TestSource(t *testing.T) { t.Log("source frontend") }
`)
	opt.Frontend = "source"
	opt.Trace = func(phase string, _ time.Duration) {
		if strings.HasPrefix(phase, "kernel-") {
			t.Errorf("source mode started %s", phase)
		}
	}
	var output bytes.Buffer
	code, err := driver.Test(opt, driver.TestOptions{Verbose: true}, &output)
	if err != nil || code != 0 || !strings.Contains(output.String(), "PASS: TestSource") {
		t.Fatalf("code %d, error %v, output %q", code, err, output.String())
	}
}

func TestNativeAssetFrontendExecution(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native UDP acceptance requires macOS or Linux")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nativefrontend\n\ngo 1.27.0\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("errors"; "fmt"; "net"; "syscall"; "time")
func main() {
 listener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
 if err != nil { panic(err) }; defer listener.Close()
 connection, err := net.DialUDP("udp4", nil, listener.LocalAddr().(*net.UDPAddr))
 if err != nil { panic(err) }; defer connection.Close()
 if n, err := connection.Write(make([]byte, 65536)); n != 0 || !errors.Is(err, syscall.EMSGSIZE) { panic(fmt.Sprint(n, err)) }
 if n, err := connection.Write([]byte("ok")); n != 2 || err != nil { panic(fmt.Sprint(n, err)) }
 listener.SetReadDeadline(time.Now().Add(time.Second))
 var data [2]byte
 if n, _, err := listener.ReadFromUDP(data[:]); n != 2 || err != nil || string(data[:]) != "ok" { panic(fmt.Sprint(n, err, data)) }
 fmt.Println("native UDP frontend passes")
}
`)
	for _, frontend := range []string{"source", "resident"} {
		t.Run(frontend, func(t *testing.T) {
			opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t), Frontend: frontend}
			var output, stderr bytes.Buffer
			code, err := driver.Run(opt, nil, &output, &stderr)
			if err != nil || code != 0 || output.String() != "native UDP frontend passes\n" || stderr.Len() != 0 {
				t.Fatalf("code %d, error %v, stdout %q, stderr %q", code, err, output.String(), stderr.String())
			}
		})
	}
}
