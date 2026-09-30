// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// Package exec runs a child process over Process of dart:io.
//
// The runtime holds the child in a table by its process id, and 7 interop
// calls carry the start, the 3 pipes, the wait, and the kill.
//
// The gaps against Go: no SysProcAttr, no ExtraFiles, no process group,
// argv[0] cannot differ from the program, a pipe of StdinPipe or StdoutPipe
// works only after Start.
package exec

import (
	"bytes"
	"context"
	"errors"
	"internal/godebug"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"syscall/dart"
	"time"
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

// ErrDot is the cause when the search of the path finds a file through a
// relative member of PATH. The caller reads it with errors.Is.
var ErrDot = errors.New("cannot run executable found relative to current directory")

// ErrWaitDelay is the cause when a child ends well and WaitDelay runs out
// before each pipe of it closes.
var ErrWaitDelay = errors.New("exec: WaitDelay expired before I/O complete")

// wrappedError names the step of the command that failed.
type wrappedError struct {
	prefix string
	err    error
}

func (w wrappedError) Error() string { return w.prefix + ": " + w.err.Error() }

func (w wrappedError) Unwrap() error { return w.err }

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

	// Cancel runs when the context of the command is done. CommandContext
	// writes the Kill method of the process there, and the caller may write
	// another action, for example a signal or a close of a pipe.
	//
	// A command that ends well after a call of Cancel gives the error of the
	// context to Wait. A Cancel that gave an error which wraps os.ErrProcessDone
	// is the 1 exception.
	//
	// A Cancel that stands on a command of Command, and not of
	// CommandContext, gives an error at Start.
	Cancel func() error

	// WaitDelay bounds 2 delays. The first is a child that does not end after
	// the context is done, and the second is a child that leaves a pipe open.
	//
	// The timer starts when the context is done, or when Wait sees the end of
	// the child, whichever comes first. At the end of the delay Wait ends the
	// child and closes each pipe of it. A child that ended well and left a pipe
	// open then gives ErrWaitDelay.
	WaitDelay time.Duration

	// ctx holds the context that CommandContext took.
	ctx context.Context

	// ctxResult carries the 1 result of watchCtx to Wait.
	ctxResult <-chan ctxResult

	// ioDone closes when the copy of each of the 3 pipes ended.
	ioDone chan struct{}

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
		// A search that gives ErrDot still names the file, and Start reports
		// the error.
		if path != "" {
			cmd.Path = path
		}
		if err != nil {
			cmd.Err = err
		}
	}
	return cmd
}

// CommandContext builds a command that the context stops. It writes the Kill
// method of the process into Cancel, and it leaves WaitDelay at 0.
func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	if ctx == nil {
		panic("nil Context")
	}
	cmd := Command(name, arg...)
	cmd.ctx = ctx
	cmd.Cancel = func() error { return cmd.Process.Kill() }
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
	if c.Cancel != nil && c.ctx == nil {
		return errors.New("exec: command with a non-nil Cancel was not created with CommandContext")
	}
	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}
	}
	argv := c.Args
	if len(argv) == 0 {
		argv = []string{c.Path}
	}
	// Go gives the child its directory before the exec, thus a relative Path
	// resolves against Dir. Process.start of Dart resolves against the
	// directory of the parent, thus the join stands here.
	path := c.Path
	if c.Dir != "" && !filepath.IsAbs(path) {
		path = filepath.Join(c.Dir, path)
	}
	full := append([]string{path}, argv[1:]...)
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
	c.ioDone = make(chan struct{})
	go func() {
		defer close(c.ioDone)
		if c.inDone != nil {
			<-c.inDone
		}
		if c.outDone != nil {
			<-c.outDone
		}
		if c.errDone != nil {
			<-c.errDone
		}
	}()
	if (c.Cancel != nil || c.WaitDelay != 0) && c.ctx != nil && c.ctx.Done() != nil {
		result := make(chan ctxResult)
		c.ctxResult = result
		go c.watchCtx(result)
	}
	return nil
}

// ctxResult carries the outcome of watchCtx. The timer stands there when
// WaitDelay started to run before the child ended.
type ctxResult struct {
	err   error
	timer *time.Timer
}

// watchCtx waits until the context is done or until Wait takes the result.
//
// A context that is done runs Cancel. A WaitDelay that then runs out ends the
// child and closes each pipe of it.
func (c *Cmd) watchCtx(result chan<- ctxResult) {
	select {
	case result <- ctxResult{}:
		return
	case <-c.ctx.Done():
	}
	var err error
	if c.Cancel != nil {
		if cancelErr := c.Cancel(); cancelErr == nil {
			err = c.ctx.Err()
		} else if !errors.Is(cancelErr, os.ErrProcessDone) {
			err = wrappedError{prefix: "exec: canceling Cmd", err: cancelErr}
		}
	}
	if c.WaitDelay == 0 {
		result <- ctxResult{err: err}
		return
	}
	timer := time.NewTimer(c.WaitDelay)
	select {
	case result <- ctxResult{err: err, timer: timer}:
		return
	case <-timer.C:
	}
	if killErr := c.Process.Kill(); killErr != nil &&
		!errors.Is(killErr, os.ErrProcessDone) {
		err = wrappedError{prefix: "exec: killing Cmd", err: killErr}
	}
	select {
	case <-c.ioDone:
	default:
		dart.ProcCloseIO(c.Process.Pid)
		<-c.ioDone
		if err == nil {
			err = ErrWaitDelay
		}
	}
	result <- ctxResult{err: err}
}

// awaitCopies waits for the copy of each pipe. A WaitDelay that runs out closes
// the pipes of the child. An orphan that holds a pipe thus stops no caller.
func (c *Cmd) awaitCopies(timer *time.Timer) error {
	if c.ioDone == nil {
		return nil
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	if timer == nil {
		if c.WaitDelay == 0 {
			<-c.ioDone
			return nil
		}
		select {
		case <-c.ioDone:
			return nil
		default:
		}
		timer = time.NewTimer(c.WaitDelay)
	}
	select {
	case <-c.ioDone:
		return nil
	case <-timer.C:
		dart.ProcCloseIO(c.Process.Pid)
		<-c.ioDone
		return ErrWaitDelay
	}
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
	if err == nil && !state.Success() {
		err = &ExitError{ProcessState: state}
	}
	c.ProcessState = state
	var timer *time.Timer
	if c.ctxResult != nil {
		watch := <-c.ctxResult
		timer = watch.timer
		if err == nil && watch.err != nil {
			err = watch.err
		}
	}
	if copyErr := c.awaitCopies(timer); err == nil {
		err = copyErr
	}
	dart.ProcCloseIO(c.Process.Pid)
	return err
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

// Environ gives the environment of the command as it stands now. A nil Env
// reads the environment of this program, and a Dir that is not empty adds PWD.
// A name that stands 2 times keeps the last value.
func (c *Cmd) Environ() []string {
	env, _ := c.environ()
	return env
}

func (c *Cmd) environ() ([]string, error) {
	var err error
	env := c.Env
	if env == nil {
		env = os.Environ()
		if c.Dir != "" {
			// PWD names the working directory on a POSIX platform, and the
			// child runs in another directory than this program.
			if pwd, absErr := filepath.Abs(c.Dir); absErr == nil {
				env = append(env, "PWD="+pwd)
			} else {
				err = absErr
			}
		}
	}
	out, dedupErr := dedupEnv(env)
	if err == nil {
		err = dedupErr
	}
	return out, err
}

// dedupEnv drops each name that stands 2 times, and it keeps the last value of
// such a name. An entry that holds a zero byte goes, and it gives an error.
func dedupEnv(env []string) ([]string, error) {
	return dedupEnvCase(runtime.GOOS == "windows", runtime.GOOS == "plan9", env)
}

func dedupEnvCase(caseInsensitive, nulOK bool, env []string) ([]string, error) {
	var err error
	out := make([]string, 0, len(env))
	saw := make(map[string]bool, len(env))
	// The walk runs backwards, thus the last value of a name is the first one
	// that the walk meets.
	for n := len(env); n > 0; n-- {
		item := env[n-1]
		if !nulOK && strings.IndexByte(item, 0) != -1 {
			err = errors.New("exec: environment variable contains NUL")
			continue
		}
		i := strings.Index(item, "=")
		if i == 0 {
			i = strings.Index(item[1:], "=") + 1
		}
		if i < 0 {
			if item != "" {
				out = append(out, item)
			}
			continue
		}
		name := item[:i]
		if caseInsensitive {
			name = strings.ToLower(name)
		}
		if saw[name] {
			continue
		}
		saw[name] = true
		out = append(out, item)
	}
	for i := 0; i < len(out)/2; i++ {
		j := len(out) - i - 1
		out[i], out[j] = out[j], out[i]
	}
	return out, err
}

var execerrdot = godebug.New("execerrdot")

// LookPath searches PATH for an executable file. A name with a separator
// skips the search.
//
// A member of PATH that is relative gives the path and an error that errors.Is
// reads as ErrDot unless GODEBUG=execerrdot=0 permits that lookup.
func LookPath(file string) (string, error) {
	switch file {
	case "", ".", "..":
		return "", &Error{Name: file, Err: ErrNotFound}
	}
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
			if !filepath.IsAbs(path) {
				if execerrdot.Value() != "0" {
					return path, &Error{Name: file, Err: ErrDot}
				}
				execerrdot.IncNonDefault()
			}
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
		return syscall.EISDIR
	}
	if dart.HasNativeStat() {
		code := dart.NativeExecutableAccess(file)
		if code == 0 {
			return nil
		}
		if code != int(syscall.ENOSYS) && code != int(syscall.EPERM) {
			return syscall.Errno(code)
		}
	}
	if d.Mode()&0111 != 0 {
		return nil
	}
	return fs.ErrPermission
}
