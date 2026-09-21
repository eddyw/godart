package main

import (
	"syscall"
	"time"
)

func rawModTime(s *syscall.Stat_t) time.Time {
	return time.Unix(s.Mtim.Sec, s.Mtim.Nsec)
}
