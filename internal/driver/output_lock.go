package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func (opt *Options) lockOutput() (func(), error) {
	if opt.outputLocked {
		return func() {}, nil
	}
	directory := filepath.Join(opt.Out, ".gart")
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(directory, "output.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("output directory is in use: %s: %w", opt.Out, err)
	}
	opt.outputLocked = true
	return func() { file.Close() }, nil
}
