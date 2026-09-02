//go:build dart

package net

import "context"

// A ListenConfig opens a listener or a datagram socket with a context. The
// context ends the wait for the bind.
//
// Go holds 3 more items on the type. `Control` takes a `syscall.RawConn`, and
// the surface holds no `syscall` package, because `dart:io` gives no file
// descriptor. `KeepAlive` and `KeepAliveConfig` each write a socket option
// that no intrinsic of `syscall/dart` reaches. `MultipathTCP` and
// `SetMultipathTCP` name a transport of Linux that Dart never opens. The
// overlay holds none of the 5, because a field that reads no value lies.
type ListenConfig struct{}

// listenResult holds the 2 results of a bind that runs on a goroutine.
type listenResult struct {
	listener Listener
	packet   PacketConn
	cause    error
}

// Listen opens a listener on an address. The network is tcp or unix, exactly
// as in Listen of this package.
//
// The bind runs on a goroutine, thus the select takes the first of the 2. A
// bind that ends after the context closes its own listener.
func (lc *ListenConfig) Listen(ctx context.Context, network, address string) (Listener, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	done := make(chan listenResult, 1)
	go func() {
		l, cause := Listen(network, address)
		done <- listenResult{listener: l, cause: cause}
	}()
	select {
	case r := <-done:
		return r.listener, r.cause
	case <-ctx.Done():
		go func() {
			if r := <-done; r.listener != nil {
				r.listener.Close()
			}
		}()
		return nil, ctx.Err()
	}
}

// ListenPacket opens a datagram socket on an address. The network is udp,
// exactly as in ListenPacket of this package.
func (lc *ListenConfig) ListenPacket(ctx context.Context, network, address string) (PacketConn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	done := make(chan listenResult, 1)
	go func() {
		p, cause := ListenPacket(network, address)
		done <- listenResult{packet: p, cause: cause}
	}()
	select {
	case r := <-done:
		return r.packet, r.cause
	case <-ctx.Done():
		go func() {
			if r := <-done; r.packet != nil {
				r.packet.Close()
			}
		}()
		return nil, ctx.Err()
	}
}
