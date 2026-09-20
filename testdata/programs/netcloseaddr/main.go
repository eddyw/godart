package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func pair(listener net.Listener) (net.Conn, net.Conn) {
	accepted := make(chan net.Conn)
	go func() {
		conn, err := listener.Accept()
		must(err)
		accepted <- conn
	}()
	client, err := net.Dial(listener.Addr().Network(), listener.Addr().String())
	must(err)
	return client, <-accepted
}

func main() {
	dir, err := os.MkdirTemp("", "netaddr-")
	must(err)
	defer os.RemoveAll(dir)
	for _, test := range []struct{ name, network, address string }{
		{"tcp4", "tcp", "127.0.0.1:0"},
		{"tcp6", "tcp", "[::1]:0"},
		{"unix", "unix", filepath.Join(dir, "socket")},
	} {
		listener, err := net.Listen(test.network, test.address)
		must(err)
		client, peer := pair(listener)
		local, remote := client.LocalAddr().String(), client.RemoteAddr().String()
		must(client.(interface{ CloseWrite() error }).CloseWrite())
		_, err = io.ReadAll(peer)
		must(err)
		must(peer.Close())
		_, err = io.ReadAll(client)
		must(err)
		time.Sleep(10 * time.Millisecond)
		fmt.Println(test.name, "peer closed", client.LocalAddr().String() == local, client.RemoteAddr().String() == remote)
		must(client.Close())
		fmt.Println(test.name, "local closed", client.LocalAddr().String() == local, client.RemoteAddr().String() == remote)
		next, nextPeer := pair(listener)
		client.SetReadDeadline(time.Unix(1, 0))
		_, err = client.Read(make([]byte, 1))
		fmt.Println(test.name, "closed read", errors.Is(err, net.ErrClosed))
		fmt.Println(test.name, "new connection", client.LocalAddr().String() == local, client.RemoteAddr().String() == remote)
		must(next.Close())
		must(nextPeer.Close())
		must(listener.Close())
	}
}
