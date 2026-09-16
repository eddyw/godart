package driver_test

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// runLimit bounds each external command of the suite. A program of the test
// directory of Go can loop for ever under the block model, and the suite runs
// more than 1000 such programs. Without a limit such a program holds its
// process and its pipes after the test binary ends.
const runLimit = 5 * time.Minute

// spawnSlots bounds how many external commands the suite runs at the same
// time. 1 subtest of TestPrograms or of TestRunFilesOfGoTest spawns go list,
// go run, dart pub get, and dart run. Go gives each parallel subtest a slot of
// GOMAXPROCS, thus 16 cores let 16 subtests hold about 48 tool chain
// processes. Each go command of a module opens the source that it reads, and
// a full go mod tidy of this module opens more than 5000 files.
//
// 8 leaves room for the editor. gopls runs go mod tidy and go list over each
// module that appears, and the suite writes more than 1000 temporary modules.
var spawnSlots = make(chan struct{}, 8)

// takeSlot waits for a slot and gives the release of it.
func takeSlot() func() {
	spawnSlots <- struct{}{}
	return func() { <-spawnSlots }
}

// output runs a command under a deadline and under the slot limit, and it
// gives the combined output. ok reports whether the command ended inside the
// deadline.
//
// The command runs in a process group of its own, and the deadline kills that
// whole group. `go run` starts the program as a child of its own, thus a kill
// of `go run` alone leaves that program alive with its pipes open.
func output(dir string, env []string, limit time.Duration, name string, args ...string) (text []byte, err error, ok bool) {
	defer takeSlot()()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 10 * time.Second
	text, err = cmd.CombinedOutput()
	return text, err, ctx.Err() == nil
}

func traceBuild(t *testing.T) func(string, time.Duration) {
	return func(phase string, elapsed time.Duration) {
		t.Logf("phase=%s duration=%.3fs", phase, elapsed.Seconds())
	}
}
