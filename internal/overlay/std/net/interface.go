//go:build dart

package net

import (
	"errors"
	"strconv"
	"strings"
	"syscall/dart"
)

// This file holds the network interface table.
//
// `NetworkInterface.list` of `dart:io` gives 3 items of each interface: the
// name, the index, and each address with its prefix length. It gives nothing
// else.
//
// Thus the Interface type of this overlay holds Index and Name alone. Go also
// holds MTU, HardwareAddr, and Flags, and Dart reports none of the 3. A field
// that always reads 0 lies about the host, thus the overlay holds no such
// field. `MulticastAddrs` stays out for the same cause: Dart lists no joined
// multicast group.
//
// A web target holds no interface table. Go under GOOS=js gives an empty
// table and no error, and the web form of the runtime does the same.

// The 4 errors of the interface table. Go wraps each 1 in an OpError. That
// error holds the operation "route" and the network "ip+net", thus each text
// below matches the text of Go.
var (
	errInvalidInterface      = errors.New("route ip+net: invalid network interface")
	errInvalidInterfaceIndex = errors.New("route ip+net: invalid network interface index")
	errInvalidInterfaceName  = errors.New("route ip+net: invalid network interface name")
	errNoSuchInterface       = errors.New("route ip+net: no such network interface")
)

// An Interface names 1 network interface of the host.
//
// Go also holds MTU, HardwareAddr, and Flags. `dart:io` reports none of the
// 3, thus this overlay holds no such field. The comment at the top of the
// file holds the cause.
type Interface struct {
	// Index is a positive integer that starts at 1. The value 0 names no
	// interface.
	Index int
	// Name is the name of the interface, for example "en0" or "lo0".
	Name string
}

// Interfaces gives each network interface of the host.
func Interfaces() ([]Interface, error) {
	table, _, err := interfaceTable()
	return table, err
}

// InterfaceByIndex gives the interface of an index. Go reports the first
// interface of the index on a system that holds an alias. Dart lists 1 entry
// for each index.
func InterfaceByIndex(index int) (*Interface, error) {
	if index <= 0 {
		return nil, errInvalidInterfaceIndex
	}
	table, _, err := interfaceTable()
	if err != nil {
		return nil, err
	}
	for i := range table {
		if table[i].Index == index {
			out := table[i]
			return &out, nil
		}
	}
	return nil, errNoSuchInterface
}

// InterfaceByName gives the interface of a name.
func InterfaceByName(name string) (*Interface, error) {
	if name == "" {
		return nil, errInvalidInterfaceName
	}
	table, _, err := interfaceTable()
	if err != nil {
		return nil, err
	}
	for i := range table {
		if table[i].Name == name {
			out := table[i]
			return &out, nil
		}
	}
	return nil, errNoSuchInterface
}

// Addrs gives each unicast address of the interface. Each address holds the
// dynamic type *IPNet, exactly as in Go. The mask comes from the prefix
// length that Dart reports.
func (ifi *Interface) Addrs() ([]Addr, error) {
	if ifi == nil {
		return nil, errInvalidInterface
	}
	_, addrs, err := interfaceTable()
	if err != nil {
		return nil, err
	}
	var out []Addr
	for _, a := range addrs {
		if a.index == ifi.Index {
			out = append(out, a.addr)
		}
	}
	return out, nil
}

// InterfaceAddrs gives each unicast address of each interface of the host.
// The list names no interface, thus a program that needs the pair reads
// Interfaces and Interface.Addrs.
func InterfaceAddrs() ([]Addr, error) {
	_, addrs, err := interfaceTable()
	if err != nil {
		return nil, err
	}
	out := make([]Addr, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.addr)
	}
	return out, nil
}

// interfaceAddr pairs 1 address with the index of the interface that holds
// it.
type interfaceAddr struct {
	index int
	addr  Addr
}

// interfaceTable reads the table of the host 1 time. It gives each interface
// and each address.
func interfaceTable() ([]Interface, []interfaceAddr, error) {
	names, addrs, cause := dart.Interfaces()
	if cause != "" {
		return nil, nil, errors.New("route ip+net: " + cause)
	}
	table := make([]Interface, 0, len(names))
	for _, item := range names {
		index, name, ok := splitIndex(item)
		if !ok {
			return nil, nil, errors.New("route ip+net: the host gave the interface " + item)
		}
		table = append(table, Interface{Index: index, Name: name})
	}
	list := make([]interfaceAddr, 0, len(addrs))
	for _, item := range addrs {
		index, text, ok := splitIndex(item)
		if !ok {
			return nil, nil, errors.New("route ip+net: the host gave the address " + item)
		}
		network, err := interfaceIPNet(text)
		if err != nil {
			return nil, nil, err
		}
		list = append(list, interfaceAddr{index: index, addr: network})
	}
	return table, list, nil
}

// splitIndex reads an item of the interop, in the form "<index>|<rest>".
func splitIndex(item string) (int, string, bool) {
	i := strings.IndexByte(item, '|')
	if i < 0 {
		return 0, "", false
	}
	index, err := strconv.Atoi(item[:i])
	if err != nil {
		return 0, "", false
	}
	return index, item[i+1:], true
}

// interfaceIPNet reads the address of an interface, in the form
// "<address>/<prefix>". A zone of an IPv6 address drops, because the IPNet
// type of Go holds no zone.
func interfaceIPNet(text string) (*IPNet, error) {
	i := strings.LastIndexByte(text, '/')
	if i < 0 {
		return nil, errors.New("route ip+net: the address " + text + " holds no prefix")
	}
	host, prefix := text[:i], text[i+1:]
	if z := strings.IndexByte(host, '%'); z >= 0 {
		host = host[:z]
	}
	ip := ParseIP(host)
	if ip == nil {
		return nil, errors.New("route ip+net: the address " + text + " holds no IP")
	}
	ones, err := strconv.Atoi(prefix)
	if err != nil {
		return nil, errors.New("route ip+net: the address " + text + " holds no prefix")
	}
	bits := 8 * IPv6len
	if v4 := ip.To4(); v4 != nil {
		ip = v4
		bits = 8 * IPv4len
	}
	mask := CIDRMask(ones, bits)
	if mask == nil {
		return nil, errors.New("route ip+net: the address " + text + " holds no prefix")
	}
	return &IPNet{IP: ip, Mask: mask}, nil
}
