//go:build dart

// Package exec runs a child process over Process of dart:io.
//
// The runtime holds the child in a table by its process id, and 7 interop
// calls carry the start, the 3 pipes, the wait, and the kill. Decision 204
// holds the survey, and decision 212 holds the shape.
//
// The gaps against Go: no SysProcAttr, no process group, no CommandContext,
// argv[0] cannot differ from the program, a pipe of StdinPipe or StdoutPipe
// works only after Start, and LookPath cannot read the execute bit because
// Dart names no permission bit.
package exec

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall/dart"
)

// Error joins the name that LookPath looked for and the cause.
type Error struct {
	Name string
	Err  error
}

func (e *Error) Error() string {
	return "exec: " + strconv.Quote(e.Name) + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// ErrNotFound is the cause when the search of the path finds no file.
var ErrNotFound = errors.New("executable file not found in $PATH")

// ExitError reports a process that ended with a state other than success.
type ExitError struct {
	*os.ProcessState

	// Stderr holds the standard error of the process when Output captured
	// it and no Stderr writer stood on the command.
	Stderr []byte
}

func (e *ExitError) Error() string { return e.ProcessState.String() }

// Cmd holds a command before and after its run.
type Cmd struct {
	// Path is the program. Args holds argv, with the name at index 0. The
	// start reads the program from Path, thus Args[0] changes nothing.
	Path string
	Args []string

	// Env holds each variable in the form name=value. A nil list keeps the
	// environment of this program.
	Env []string

	// Dir is the working directory of the child. An empty Dir keeps the
	// directory of this program.
	Dir string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Process      *os.Process
	ProcessState *os.ProcessState

	// Err holds the failure of LookPath inside Command, and Start gives it
	// back.
	Err error

	inPipe  bool
	outPipe bool
	errPipe bool
	inDone  chan struct{}
	outDone chan struct{}
	errDone chan struct{}
}

// Command builds a command. A name without a separator goes through LookPath,
// and a failure of that search lands in Err.
func Command(name string, arg ...string) *Cmd {
	cmd := &Cmd{
		Path: name,
		Args: append([]string{name}, arg...),
	}
	if filepath.Base(name) == name {
		path, err := LookPath(name)
		if err != nil {
			cmd.Err = err
		} else {
			cmd.Path = path
		}
	}
	return cmd
}

func (c *Cmd) String() string {
	b := new(strings.Builder)
	b.WriteString(c.Path)
	for _, a := range c.Args[1:] {
		b.WriteByte(' ')
		b.WriteString(a)
	}
	return b.String()
}

// Start starts the child and returns. Wait collects the state.
func (c *Cmd) Start() error {
	if c.Process != nil {
		return errors.New("exec: already started")
	}
	if c.Err != nil {
		return c.Err
	}
	argv := c.Args
	if len(argv) == 0 {
		argv = []string{c.Path}
	}
	full := append([]string{c.Path}, argv[1:]...)
	pid, cause := dart.ProcStart(full, c.Dir, c.Env, c.Env != nil)
	if cause != "" {
		return &Error{Name: c.Path, Err: errors.New(cause)}
	}
	c.Process = &os.Process{Pid: pid}
	if !c.inPipe {
		if c.Stdin == nil {
			dart.ProcCloseStdin(pid)
		} else {
			c.inDone = make(chan struct{})
			go func() {
				defer close(c.inDone)
				feed(pid, c.Stdin)
			}()
		}
	}
	if !c.outPipe {
		c.outDone = make(chan struct{})
		go func() {
			defer close(c.outDone)
			drain(pid, false, c.Stdout)
		}()
	}
	if !c.errPipe {
		c.errDone = make(chan struct{})
		go func() {
			defer close(c.errDone)
			drain(pid, true, c.Stderr)
		}()
	}
	return nil
}

// feed copies the reader into the standard input of the child, and it closes
// that input at the end. A write failure also ends the copy, because the
// child closed its side.
func feed(pid int, r io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if _, cause := dart.ProcWrite(pid, buf[:n]); cause != "" {
				break
			}
		}
		if err != nil {
			break
		}
	}
	dart.ProcCloseStdin(pid)
}

// drain copies 1 output of the child into the writer until the stream ends.
// A nil writer drops the bytes, because an unread pipe would stop the child
// when the pipe fills.
func drain(pid int, errStream bool, w io.Writer) {
	buf := make([]byte, 32*1024)
	for {
		var n int
		var cause string
		if errStream {
			n, cause = dart.ProcReadErr(pid, buf)
		} else {
			n, cause = dart.ProcRead(pid, buf)
		}
		if n > 0 && w != nil {
			if _, err := w.Write(buf[:n]); err != nil {
				return
			}
		}
		if cause != "" || n == 0 {
			return
		}
	}
}

// Wait waits for the end of the child and for the copies of the 3 pipes.
func (c *Cmd) Wait() error {
	if c.Process == nil {
		return errors.New("exec: not started")
	}
	if c.ProcessState != nil {
		return errors.New("exec: Wait was already called")
	}
	state, err := c.Process.Wait()
	if err != nil {
		return err
	}
	if c.outDone != nil {
		<-c.outDone
	}
	if c.errDone != nil {
		<-c.errDone
	}
	if c.inDone != nil {
		<-c.inDone
	}
	c.ProcessState = state
	if !state.Success() {
		return &ExitError{ProcessState: state}
	}
	return nil
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Output runs the command and gives its standard output. An ExitError of the
// run holds the standard error when no Stderr writer stands on the command.
func (c *Cmd) Output() ([]byte, error) {
	if c.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	var stdout bytes.Buffer
	c.Stdout = &stdout
	capture := c.Stderr == nil
	var stderr bytes.Buffer
	if capture {
		c.Stderr = &stderr
	}
	err := c.Run()
	if err != nil && capture {
		var ee *ExitError
		if errors.As(err, &ee) {
			ee.Stderr = stderr.Bytes()
		}
	}
	return stdout.Bytes(), err
}

// CombinedOutput runs the command and gives its 2 outputs as 1 list.
func (c *Cmd) CombinedOutput() ([]byte, error) {
	if c.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.Stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	err := c.Run()
	return b.Bytes(), err
}

// StdinPipe gives a writer onto the standard input of the child. A write
// works only after Start, because the pipe of the system opens there.
func (c *Cmd) StdinPipe() (io.WriteCloser, error) {
	if c.Stdin != nil {
		return nil, errors.New("exec: Stdin already set")
	}
	if c.Process != nil {
		return nil, errors.New("exec: StdinPipe after process started")
	}
	c.inPipe = true
	return &stdinPipe{c: c}, nil
}

type stdinPipe struct{ c *Cmd }

func (w *stdinPipe) Write(p []byte) (int, error) {
	if w.c.Process == nil {
		return 0, errors.New("exec: not started")
	}
	n, cause := dart.ProcWrite(w.c.Process.Pid, p)
	if cause != "" {
		return n, errors.New(cause)
	}
	return n, nil
}

func (w *stdinPipe) Close() error {
	if w.c.Process == nil {
		return errors.New("exec: not started")
	}
	if cause := dart.ProcCloseStdin(w.c.Process.Pid); cause != "" {
		return errors.New(cause)
	}
	return nil
}

// StdoutPipe gives a reader of the standard output of the child. A read
// works only after Start. Read each byte before Wait, as Go asks too.
func (c *Cmd) StdoutPipe() (io.ReadCloser, error) {
	if c.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.Process != nil {
		return nil, errors.New("exec: StdoutPipe after process started")
	}
	c.outPipe = true
	return &procReader{c: c}, nil
}

// StderrPipe gives a reader of the standard error of the child.
func (c *Cmd) StderrPipe() (io.ReadCloser, error) {
	if c.Stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	if c.Process != nil {
		return nil, errors.New("exec: StderrPipe after process started")
	}
	c.errPipe = true
	return &procReader{c: c, errStream: true}, nil
}

type procReader struct {
	c         *Cmd
	errStream bool
}

func (r *procReader) Read(p []byte) (int, error) {
	if r.c.Process == nil {
		return 0, errors.New("exec: not started")
	}
	var n int
	var cause string
	if r.errStream {
		n, cause = dart.ProcReadErr(r.c.Process.Pid, p)
	} else {
		n, cause = dart.ProcRead(r.c.Process.Pid, p)
	}
	if cause != "" {
		return n, errors.New(cause)
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (r *procReader) Close() error { return nil }

// LookPath searches the directories of the variable PATH for the file. A name
// with a separator skips the search. Dart names no permission bit, thus a
// plain file passes without the check of the execute bit.
func LookPath(file string) (string, error) {
	if strings.Contains(file, "/") {
		if err := findExecutable(file); err != nil {
			return "", &Error{Name: file, Err: err}
		}
		return file, nil
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			dir = "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(path); err == nil {
			return path, nil
		}
	}
	return "", &Error{Name: file, Err: ErrNotFound}
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if d.IsDir() {
		return fs.ErrPermission
	}
	return nil
}
