package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type listener interface {
	net.Listener
	SetDeadline(time.Time) error
}

func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func deadlineError(l listener, conn net.Conn, err error) {
	require(conn == nil, "timeout returned a connection")
	require(errors.Is(err, os.ErrDeadlineExceeded), "timeout lacks deadline identity")
	op, ok := err.(*net.OpError)
	require(ok && op.Op == "accept" && op.Net == l.Addr().Network(), "invalid accept error")
	require(op.Addr.String() == l.Addr().String() && op.Source == nil, "invalid accept error address")
	require(op.Timeout() && op.Temporary(), "invalid timeout properties")
}

func main() {
	require(os.IsTimeout(os.ErrDeadlineExceeded), "OS deadline lacks timeout identity")
	var tcp *net.TCPListener
	var unix *net.UnixListener
	for _, l := range []listener{tcp, new(net.TCPListener), unix, new(net.UnixListener)} {
		require(errors.Is(l.SetDeadline(time.Time{}), syscall.EINVAL), "invalid listener deadline")
	}
	dir, err := os.MkdirTemp("", "gart-listener-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	for _, network := range []string{"tcp", "unix"} {
		address := "127.0.0.1:0"
		if network == "unix" {
			address = filepath.Join(dir, "socket")
		}
		l, err := net.Listen(network, address)
		if err != nil {
			panic(err)
		}
		exercise(l.(listener))
		fmt.Println(network, "deadlines pass")
	}
}

func exercise(l listener) {
	defer l.Close()
	set := func(t time.Time) {
		if err := l.SetDeadline(t); err != nil {
			panic(err)
		}
	}
	set(time.Now().Add(-time.Second))
	conn, err := l.Accept()
	deadlineError(l, conn, err)

	client, err := net.Dial(l.Addr().Network(), l.Addr().String())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	client.Write([]byte{42})
	time.Sleep(20 * time.Millisecond)
	for range 3 {
		conn, err = l.Accept()
		deadlineError(l, conn, err)
	}
	set(time.Time{})
	conn, err = l.Accept()
	if err != nil {
		panic(err)
	}
	var data [1]byte
	_, err = io.ReadFull(conn, data[:])
	require(err == nil && data[0] == 42, "timeout lost the next connection")
	conn.Close()

	const waiters = 4
	errorsOut := make(chan error, waiters)
	for range waiters {
		go func() {
			conn, err := l.Accept()
			deadlineError(l, conn, err)
			errorsOut <- err
		}()
	}
	time.Sleep(20 * time.Millisecond)
	set(time.Now().Add(20 * time.Millisecond))
	for range waiters {
		<-errorsOut
	}

	set(time.Now().Add(50 * time.Millisecond))
	go func() {
		conn, err := l.Accept()
		if conn != nil {
			conn.Close()
		}
		errorsOut <- err
	}()
	time.Sleep(10 * time.Millisecond)
	set(time.Time{})
	select {
	case <-errorsOut:
		panic("removed deadline ended accept")
	case <-time.After(100 * time.Millisecond):
	}
	l.Close()
	require(errors.Is(<-errorsOut, net.ErrClosed), "close did not release accept")
	require(errors.Is(l.SetDeadline(time.Time{}), net.ErrClosed), "closed deadline lacks closed identity")
}
