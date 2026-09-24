package main

import (
	"fmt"
	"syscall"
)

func recovered(call func()) {
	defer func() { fmt.Println("panic", recover()) }()
	call()
}

func main() {
	for _, n := range []int64{
		-1 << 63, -1<<63 + 1, -1<<63 + 999,
		-1 << 53, -1_000_000_001, -1_000_000_000, -999_999_999,
		-1_000_001, -1_000_000, -999_999, -1001, -1000, -999, -1,
		0, 1, 999, 1000, 1001, 999_999, 1_000_000, 1_000_001,
		999_999_999, 1_000_000_000, 1_000_000_001, 1 << 53,
		1<<63 - 1000, 1<<63 - 999, 1<<63 - 1,
	} {
		ts := syscall.NsecToTimespec(n)
		sec, nsec := ts.Unix()
		fmt.Println("timespec", n, ts.Sec, ts.Nsec, sec, nsec, ts.Nano(), syscall.TimespecToNsec(ts))
		tv := syscall.NsecToTimeval(n)
		sec, nsec = tv.Unix()
		fmt.Println("timeval", n, tv.Sec, tv.Usec, sec, nsec, tv.Nano(), syscall.TimevalToNsec(tv))
	}
	for _, ts := range []syscall.Timespec{{Sec: -2, Nsec: -1}, {Sec: 2, Nsec: 1_000_000_001}} {
		sec, nsec := ts.Unix()
		fmt.Println("raw timespec", sec, nsec, ts.Nano(), syscall.TimespecToNsec(ts))
	}
	for _, tv := range []syscall.Timeval{{Sec: -2, Usec: -1}, {Sec: 2, Usec: 1_000_001}} {
		sec, nsec := tv.Unix()
		fmt.Println("raw timeval", sec, nsec, tv.Nano(), syscall.TimevalToNsec(tv))
	}
	var ts *syscall.Timespec
	var tv *syscall.Timeval
	recovered(func() { ts.Nano() })
	recovered(func() { tv.Nano() })
	recovered(func() { tv.Unix() })
}
