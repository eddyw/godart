// Command netvalue exercises the value types and the errors of net that need
// no socket: Buffers, HardwareAddr, ParseMAC, Flags, DNSConfigError, and the
// 3 text methods of IP. It also proves that a closed listener gives an error
// that wraps net.ErrClosed.
//
// Each line names a value that no machine changes.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
)

// buffers runs the 2 methods of net.Buffers.
func buffers() {
	var out bytes.Buffer
	v := net.Buffers{[]byte("the quick "), []byte("brown "), []byte("fox")}
	n, err := v.WriteTo(&out)
	fmt.Println("write to:", n, err, out.String(), len(v))

	r := net.Buffers{[]byte("abc"), []byte("de")}
	p := make([]byte, 4)
	n2, err := r.Read(p)
	fmt.Println("read 1:", n2, err, string(p[:n2]))
	n2, err = r.Read(p)
	fmt.Println("read 2:", n2, err == io.EOF, string(p[:n2]))
	n2, err = r.Read(p)
	fmt.Println("read 3:", n2, err == io.EOF)

	// A writer that fails stops the write and drops the bytes that went out.
	short := net.Buffers{[]byte("one"), []byte("two")}
	n, err = short.WriteTo(failWriter{})
	fmt.Println("write fail:", n, err, len(short))
}

// failWriter gives an error for each write.
type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, errors.New("no room") }

// hardware runs HardwareAddr and ParseMAC.
func hardware() {
	forms := []string{
		"00:00:5e:00:53:01",
		"02:00:5e:10:00:00:00:01",
		"00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01",
		"00-00-5e-00-53-01",
		"0000.5e00.5301",
		"0200.5e10.0000.0001",
		"00005e005301",
		"00:00:5e:00:53",
		"zz:00:5e:00:53:01",
		"",
	}
	for _, form := range forms {
		hw, err := net.ParseMAC(form)
		fmt.Printf("mac %q -> %q %v %d\n", form, hw.String(), err, len(hw))
	}
	fmt.Println("empty:", net.HardwareAddr(nil).String() == "")
	fmt.Println("one byte:", net.HardwareAddr([]byte{0xab}).String())
}

// flags runs the 6 constants and the String method.
func flags() {
	each := []net.Flags{
		net.FlagUp,
		net.FlagBroadcast,
		net.FlagLoopback,
		net.FlagPointToPoint,
		net.FlagMulticast,
		net.FlagRunning,
	}
	for _, f := range each {
		fmt.Printf("flag %d %v\n", uint(f), f)
	}
	fmt.Println("none:", net.Flags(0).String())
	fmt.Println("many:", net.FlagUp|net.FlagLoopback|net.FlagRunning)
}

// text runs AppendText, MarshalText, and UnmarshalText of IP.
func text() {
	v4 := net.ParseIP("192.168.0.1")
	b, err := v4.AppendText([]byte("ip="))
	fmt.Println("append:", string(b), err)
	t, err := v4.MarshalText()
	fmt.Println("marshal:", string(t), err)

	v6 := net.ParseIP("2001:db8::1")
	t, err = v6.MarshalText()
	fmt.Println("marshal 6:", string(t), err)

	empty := net.IP(nil)
	b, err = empty.AppendText([]byte("x"))
	fmt.Println("append empty:", string(b), err)
	t, err = empty.MarshalText()
	fmt.Println("marshal empty:", len(t), err)

	bad := net.IP([]byte{1, 2, 3})
	b, err = bad.AppendText(nil)
	fmt.Println("append bad:", len(b), err)
	t, err = bad.MarshalText()
	fmt.Println("marshal bad:", t == nil, err)

	var back net.IP
	fmt.Println("unmarshal:", back.UnmarshalText([]byte("10.1.2.3")), back)
	fmt.Println("unmarshal 6:", back.UnmarshalText([]byte("::1")), back)
	fmt.Println("unmarshal empty:", back.UnmarshalText([]byte("")), back == nil)
	fmt.Println("unmarshal bad:", back.UnmarshalText([]byte("nope")))
}

// failures runs DNSConfigError, ErrClosed, and ErrWriteToConnected.
func failures() {
	e := &net.DNSConfigError{Err: errors.New("open /etc/resolv.conf: no such file")}
	fmt.Println("dns config:", e.Error())
	fmt.Println("dns config flags:", e.Timeout(), e.Temporary())
	fmt.Println("dns config unwrap:", e.Unwrap() != nil)
	var netErr net.Error = e
	fmt.Println("dns config is a net.Error:", netErr.Timeout())

	fmt.Println("write to connected:", net.ErrWriteToConnected.Error())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	if err := ln.Close(); err != nil {
		fmt.Println("close:", err)
		return
	}
	_, err = ln.Accept()
	fmt.Println("accept after close:", errors.Is(err, net.ErrClosed))
	fmt.Println("close twice:", errors.Is(ln.Close(), net.ErrClosed))
}

func main() {
	buffers()
	hardware()
	flags()
	text()
	failures()
}
