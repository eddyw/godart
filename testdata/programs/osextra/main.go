// Command osextra exercises the names that the packages os and os/exec gained
// on top of the first surface.
//
// The names cover the mode, the owner, the time, the size, the hard link, the
// copy of a file system, and the context of a command.
//
// Each line holds a result that does not change from 1 run to the next. The
// program works inside a temporary directory, and it prints no absolute path,
// no time of the moment, and no process id.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	base := os.TempDir() + "/gart_osextra"
	os.RemoveAll(base)
	if err := os.MkdirAll(base, 0o755); err != nil {
		fmt.Println("mkdir", err)
		return
	}
	if err := os.Chdir(base); err != nil {
		fmt.Println("chdir", err)
		return
	}

	mode()
	owner()
	times()
	size()
	links()
	copyTree()
	streams()
	deadlines()
	root()
	commands()
}

// mode covers Chmod and File.Chmod.
func mode() {
	os.WriteFile("mode.txt", []byte("mode\n"), 0o644)
	fmt.Println("chmod:", os.Chmod("mode.txt", 0o400) == nil)

	// A file of the mode 0o400 takes no write. A run under the root user opens
	// the file, and the 2 toolchains then agree on false.
	w, err := os.OpenFile("mode.txt", os.O_WRONLY, 0)
	fmt.Println("chmod read only:", err != nil, os.IsPermission(err))
	if err == nil {
		w.Close()
	}

	fmt.Println("chmod back:", os.Chmod("mode.txt", 0o644) == nil)
	w, err = os.OpenFile("mode.txt", os.O_WRONLY, 0)
	fmt.Println("chmod write:", err == nil)
	if err == nil {
		w.Close()
	}

	err = os.Chmod("missing.txt", 0o644)
	fmt.Println("chmod missing:", err != nil, os.IsNotExist(err))

	f, err := os.Create("fmode.txt")
	if err != nil {
		fmt.Println("create", err)
		return
	}
	fmt.Println("file chmod:", f.Chmod(0o600) == nil)
	f.Close()
}

// owner covers Chown, Lchown, and File.Chown. A pair of -1 changes nothing,
// and a run of any user gives the same result.
func owner() {
	fmt.Println("chown:", os.Chown("mode.txt", -1, -1) == nil)
	fmt.Println("chown missing:", os.IsNotExist(os.Chown("missing.txt", -1, -1)))

	os.Remove("mode.link")
	os.Symlink("mode.txt", "mode.link")
	fmt.Println("lchown:", os.Lchown("mode.link", -1, -1) == nil)
	fmt.Println("lchown missing:", os.IsNotExist(os.Lchown("missing.link", -1, -1)))

	f, err := os.Open("mode.txt")
	if err != nil {
		fmt.Println("open", err)
		return
	}
	fmt.Println("file chown:", f.Chown(-1, -1) == nil)
	f.Close()
}

// when is the time that the tests of Chtimes write. The Dart side holds the
// millisecond, thus the value stands on a whole second.
var when = time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)

// times covers Chtimes.
func times() {
	fmt.Println("chtimes:", os.Chtimes("mode.txt", when, when) == nil)
	info, err := os.Stat("mode.txt")
	if err != nil {
		fmt.Println("stat", err)
		return
	}
	fmt.Println("chtimes mod:", info.ModTime().Unix())

	// A zero time leaves that 1 of the 2 alone.
	fmt.Println("chtimes zero:", os.Chtimes("mode.txt", time.Time{}, time.Time{}) == nil)
	again, _ := os.Stat("mode.txt")
	fmt.Println("chtimes kept:", again.ModTime().Unix())

	fmt.Println("chtimes missing:",
		os.IsNotExist(os.Chtimes("missing.txt", when, when)))
}

// size covers Truncate and File.Truncate.
func size() {
	os.WriteFile("cut.txt", []byte("abcdefghij"), 0o644)
	fmt.Println("truncate:", os.Truncate("cut.txt", 4) == nil)
	body, _ := os.ReadFile("cut.txt")
	fmt.Printf("truncate body: %q\n", body)

	fmt.Println("truncate grow:", os.Truncate("cut.txt", 6) == nil)
	body, _ = os.ReadFile("cut.txt")
	fmt.Printf("truncate grown: %q\n", body)

	fmt.Println("truncate missing:", os.IsNotExist(os.Truncate("missing.txt", 1)))
	_, err := os.Stat("missing.txt")
	fmt.Println("truncate made nothing:", os.IsNotExist(err))

	f, err := os.OpenFile("cut.txt", os.O_RDWR, 0)
	if err != nil {
		fmt.Println("open", err)
		return
	}
	fmt.Println("file truncate:", f.Truncate(2) == nil)
	f.Close()
	body, _ = os.ReadFile("cut.txt")
	fmt.Printf("file truncate body: %q\n", body)

	// File.Chdir needs a directory, and a plain file gives an error.
	d, err := os.Create("notdir.txt")
	if err != nil {
		fmt.Println("create", err)
		return
	}
	fmt.Println("file chdir:", d.Chdir() != nil)
	d.Close()
}

// links covers Link.
func links() {
	os.Remove("link_dst.txt")
	os.WriteFile("link_src.txt", []byte("shared\n"), 0o644)
	fmt.Println("link:", os.Link("link_src.txt", "link_dst.txt") == nil)

	body, _ := os.ReadFile("link_dst.txt")
	fmt.Printf("link body: %q\n", body)

	info, err := os.Lstat("link_dst.txt")
	fmt.Println("link is no symlink:", err == nil, info.Mode()&fs.ModeSymlink == 0)

	// The 2 names hold 1 content.
	os.WriteFile("link_src.txt", []byte("changed\n"), 0o644)
	body, _ = os.ReadFile("link_dst.txt")
	fmt.Printf("link shared: %q\n", body)

	fmt.Println("link twice:", os.Link("link_src.txt", "link_dst.txt") != nil)
	fmt.Println("link missing:", os.Link("missing.txt", "link_none.txt") != nil)
}

// copyTree covers CopyFS.
func copyTree() {
	os.RemoveAll("tree")
	os.RemoveAll("copy")
	os.MkdirAll("tree/sub", 0o755)
	os.WriteFile("tree/a.txt", []byte("A"), 0o644)
	os.WriteFile("tree/sub/b.txt", []byte("B"), 0o644)

	fmt.Println("copyfs:", os.CopyFS("copy", os.DirFS("tree")) == nil)
	a, _ := os.ReadFile("copy/a.txt")
	b, _ := os.ReadFile("copy/sub/b.txt")
	fmt.Printf("copyfs body: %q %q\n", a, b)

	err := os.CopyFS("copy", os.DirFS("tree"))
	fmt.Println("copyfs again:", err != nil, errors.Is(err, fs.ErrExist))
}

// streams covers File.ReadFrom and File.WriteTo.
func streams() {
	f, err := os.Create("stream.txt")
	if err != nil {
		fmt.Println("create", err)
		return
	}
	n, err := f.ReadFrom(strings.NewReader("read from\n"))
	fmt.Println("readfrom:", n, err == nil)
	f.Close()
	body, _ := os.ReadFile("stream.txt")
	fmt.Printf("readfrom body: %q\n", body)

	r, err := os.Open("stream.txt")
	if err != nil {
		fmt.Println("open", err)
		return
	}
	var out strings.Builder
	n, err = r.WriteTo(&out)
	fmt.Println("writeto:", n, err == nil, out.String() == "read from\n")
	r.Close()
}

// deadlines covers File.SetDeadline, File.SetReadDeadline, and
// File.SetWriteDeadline. An ordinary file takes no deadline.
func deadlines() {
	f, err := os.Open("stream.txt")
	if err != nil {
		fmt.Println("open", err)
		return
	}
	fmt.Println("deadline:",
		errors.Is(f.SetDeadline(time.Time{}), os.ErrNoDeadline),
		errors.Is(f.SetReadDeadline(when), os.ErrNoDeadline),
		errors.Is(f.SetWriteDeadline(when), os.ErrNoDeadline))
	f.Close()
}

// root covers Root.Chmod, Root.Chown, Root.Lchown, Root.Chtimes, and Root.Link.
func root() {
	os.RemoveAll("rootdir")
	os.MkdirAll("rootdir", 0o755)
	os.WriteFile("rootdir/r.txt", []byte("root\n"), 0o644)

	r, err := os.OpenRoot("rootdir")
	if err != nil {
		fmt.Println("openroot", err)
		return
	}
	defer r.Close()

	fmt.Println("root chmod:", r.Chmod("r.txt", 0o600) == nil)
	fmt.Println("root chown:", r.Chown("r.txt", -1, -1) == nil)
	fmt.Println("root lchown:", r.Lchown("r.txt", -1, -1) == nil)
	fmt.Println("root chtimes:", r.Chtimes("r.txt", when, when) == nil)
	info, err := r.Stat("r.txt")
	fmt.Println("root chtimes mod:", err == nil, info.ModTime().Unix())

	fmt.Println("root link:", r.Link("r.txt", "r2.txt") == nil)
	body, err := r.ReadFile("r2.txt")
	fmt.Printf("root link body: %q %v\n", body, err == nil)

	fmt.Println("root escape:",
		r.Chmod("../mode.txt", 0o644) != nil,
		r.Chown("../mode.txt", -1, -1) != nil,
		r.Chtimes("../mode.txt", when, when) != nil,
		r.Link("../mode.txt", "out.txt") != nil)
}

// commands covers CommandContext, Cmd.Cancel, Cmd.WaitDelay, Cmd.Environ,
// ErrDot, and ErrWaitDelay.
func commands() {
	done, stop := context.WithCancel(context.Background())
	stop()
	err := exec.CommandContext(done, "echo", "hi").Run()
	fmt.Println("ctx done:", errors.Is(err, context.Canceled))

	short, drop := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer drop()
	err = exec.CommandContext(short, "sleep", "5").Run()
	fmt.Println("ctx kill:", err)
	fmt.Println("ctx deadline:", errors.Is(short.Err(), context.DeadlineExceeded))

	open, cancel := context.WithCancel(context.Background())
	defer cancel()
	slow := exec.CommandContext(open, "sleep", "5")
	slow.Cancel = func() error { return slow.Process.Signal(os.Interrupt) }
	if err := slow.Start(); err != nil {
		fmt.Println("start", err)
		return
	}
	cancel()
	fmt.Println("cancel signal:", slow.Wait())

	loose := exec.Command("echo", "x")
	loose.Cancel = func() error { return nil }
	fmt.Println("cancel without context:", loose.Start())

	// A child that ends and leaves an orphan on its pipe stops the wait, and
	// WaitDelay bounds that wait.
	hold := exec.Command("sh", "-c", "sleep 2 & echo held")
	hold.WaitDelay = 200 * time.Millisecond
	out, err := hold.Output()
	fmt.Printf("waitdelay: %q %v\n", out, errors.Is(err, exec.ErrWaitDelay))

	named := exec.Command("echo")
	named.Env = []string{"A=1", "B=2", "A=3"}
	fmt.Println("environ:", named.Environ())
	plain := exec.Command("echo")
	fmt.Println("environ default:", len(plain.Environ()) == len(os.Environ()))
	here := exec.Command("echo")
	here.Dir = "."
	env := here.Environ()
	fmt.Println("environ pwd:", strings.HasPrefix(env[len(env)-1], "PWD="))

	errDot()
}

// errDot covers ErrDot. A member of PATH that is relative gives that error.
func errDot() {
	os.MkdirAll("bin", 0o755)
	os.WriteFile("bin/gartool", []byte("#!/bin/sh\necho tool\n"), 0o755)
	os.Chmod("bin/gartool", 0o755)

	keep := os.Getenv("PATH")
	os.Setenv("PATH", "bin")
	path, err := exec.LookPath("gartool")
	fmt.Println("errdot lookpath:", path, errors.Is(err, exec.ErrDot))

	cmd := exec.Command("gartool")
	fmt.Println("errdot command:", cmd.Path, errors.Is(cmd.Err, exec.ErrDot))
	_, err = cmd.Output()
	fmt.Println("errdot run:", errors.Is(err, exec.ErrDot))
	os.Setenv("PATH", keep)
}
