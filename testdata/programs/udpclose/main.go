package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func open() net.PacketConn {
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	return c
}

func closed(err error) {
	if !errors.Is(err, net.ErrClosed) {
		panic(fmt.Sprintf("expected closed socket: %v", err))
	}
}

func main() {
	old := open()
	address := old.LocalAddr().String()
	if err := old.Close(); err != nil {
		panic(err)
	}
	current := open()
	defer current.Close()
	closed(old.SetDeadline(time.Now().Add(time.Second)))
	closed(old.SetReadDeadline(time.Now().Add(time.Second)))
	closed(old.SetWriteDeadline(time.Now().Add(time.Second)))
	if err := current.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		panic(err)
	}
	_, _, err := old.ReadFrom(make([]byte, 1))
	closed(err)
	_, err = old.WriteTo([]byte{1}, current.LocalAddr())
	closed(err)
	closed(old.Close())
	if old.LocalAddr().String() != address {
		panic("closed address changed")
	}
	if _, err := current.WriteTo([]byte{42}, current.LocalAddr()); err != nil {
		panic(err)
	}
	data := make([]byte, 1)
	n, _, err := current.ReadFrom(data)
	if err != nil || n != 1 || data[0] != 42 {
		panic(fmt.Sprintf("replacement socket differs: %v", err))
	}
	fmt.Println("closed socket stays closed")
	fmt.Println("replacement socket stays open")
}
