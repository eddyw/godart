package driver_test

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeCommandFrontendExecution(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native command acceptance requires macOS or Linux")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nativecommandfrontend\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec";"syscall")
func main(){
 file,err:=os.CreateTemp("","nativeformat");if err!=nil{panic(err)};path:=file.Name();defer os.Remove(path)
 if _,err=file.WriteString("not an executable\n");err!=nil{panic(err)};if err=file.Close();err!=nil{panic(err)};if err=os.Chmod(path,0700);err!=nil{panic(err)}
 cmd:=exec.Command(path);err=cmd.Start();fmt.Println("format",errors.Is(err,syscall.ENOEXEC),"process",cmd.Process==nil);if err==nil{cmd.Wait()}
 cmd=exec.Command("sh","-c","exit 127");err=cmd.Start();if err!=nil{panic(err)};err=cmd.Wait();var exit *exec.ExitError
 fmt.Println("exit",errors.As(err,&exit),"status",cmd.ProcessState.ExitCode(),"format",errors.Is(err,syscall.ENOEXEC))
}
`)
	const expected = "format true process true\nexit true status 127 format false\n"
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 || strings.TrimSpace(want.Text) != strings.TrimSpace(expected) {
		t.Fatalf("native command contract: %+v", want)
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
