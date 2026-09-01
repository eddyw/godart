package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func main() {
	out, err := exec.Command("echo", "hello", "world").Output()
	fmt.Printf("%q %v\n", string(out), err)

	cat := exec.Command("cat")
	cat.Stdin = strings.NewReader("feed me\n")
	var buf bytes.Buffer
	cat.Stdout = &buf
	fmt.Println(cat.Run(), buf.String() == "feed me\n")

	err = exec.Command("sh", "-c", "exit 3").Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		fmt.Println("code", ee.ExitCode(), ee.Success())
	}
	fmt.Println(err)

	_, err = exec.Command("sh", "-c", "echo oops 1>&2; exit 1").Output()
	if errors.As(err, &ee) {
		fmt.Printf("%q\n", string(ee.Stderr))
	}

	_, err = exec.Command("gart-no-such-program").Output()
	fmt.Println(err)

	pwd := exec.Command("sh", "-c", "pwd; echo $GART_EXEC_TEST")
	pwd.Dir = "/"
	pwd.Env = append(os.Environ(), "GART_EXEC_TEST=on")
	out, err = pwd.Output()
	fmt.Printf("%q %v\n", string(out), err)

	pipes := exec.Command("cat")
	in, _ := pipes.StdinPipe()
	outPipe, _ := pipes.StdoutPipe()
	if err := pipes.Start(); err != nil {
		fmt.Println(err)
		return
	}
	in.Write([]byte("through the pipe\n"))
	in.Close()
	text, _ := io.ReadAll(outPipe)
	fmt.Printf("%q %v\n", string(text), pipes.Wait())

	held := exec.Command("cat")
	hold, _ := held.StdinPipe()
	if err := held.Start(); err != nil {
		fmt.Println(err)
		return
	}
	if err := held.Process.Kill(); err != nil {
		fmt.Println(err)
	}
	fmt.Println(held.Wait())
	_ = hold

	path, err := exec.LookPath("sh")
	fmt.Println(strings.HasSuffix(path, "/sh"), err)
	fmt.Println(exec.Command("echo", "a", "b").String() != "")
}
