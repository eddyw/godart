// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"syscall/dart"
)

var (
	errInvalidInterface      = errors.New("invalid network interface")
	errInvalidInterfaceIndex = errors.New("invalid network interface index")
	errInvalidInterfaceName  = errors.New("invalid network interface name")
	errNoSuchInterface       = errors.New("no such network interface")
)

// Interface describes a network interface of the host.
type Interface struct {
	Index        int
	MTU          int
	Name         string
	HardwareAddr HardwareAddr
	Flags        Flags
}

// Interfaces gives each network interface of the host.
func Interfaces() ([]Interface, error) {
	table, err := interfaceTable()
	if err != nil {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: err}
	}
	return table, nil
}

// InterfaceByIndex gives the interface with the requested index.
func InterfaceByIndex(index int) (*Interface, error) {
	if index <= 0 {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: errInvalidInterfaceIndex}
	}
	table, err := interfaceTable()
	if err != nil {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: err}
	}
	for i := range table {
		if table[i].Index == index {
			out := table[i]
			return &out, nil
		}
	}
	return nil, &OpError{Op: "route", Net: "ip+net", Err: errNoSuchInterface}
}

// InterfaceByName gives the interface of a name.
func InterfaceByName(name string) (*Interface, error) {
	if name == "" {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: errInvalidInterfaceName}
	}
	table, err := interfaceTable()
	if err != nil {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: err}
	}
	for i := range table {
		if table[i].Name == name {
			out := table[i]
			return &out, nil
		}
	}
	return nil, &OpError{Op: "route", Net: "ip+net", Err: errNoSuchInterface}
}

// Addrs gives each unicast address of the interface. Each address holds the
// dynamic type *IPNet, exactly as in Go. The mask comes from the prefix
// length that Dart reports.
func (ifi *Interface) Addrs() ([]Addr, error) {
	if ifi == nil {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: errInvalidInterface}
	}
	addrs, err := interfaceAddresses()
	if err != nil {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: err}
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
	addrs, err := interfaceAddresses()
	if err != nil {
		return nil, &OpError{Op: "route", Net: "ip+net", Err: err}
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

func interfaceTable() ([]Interface, error) {
	names, cause := dart.InterfaceTable()
	if cause != "" {
		return nil, errors.New(cause)
	}
	var table []Interface
	for _, item := range names {
		fields := strings.SplitN(item, "|", 5)
		if len(fields) != 5 {
			return nil, errors.New("invalid interface metadata")
		}
		index, indexErr := strconv.Atoi(fields[0])
		mtu, mtuErr := strconv.Atoi(fields[1])
		flags, flagsErr := strconv.ParseUint(fields[2], 10, 32)
		if indexErr != nil || mtuErr != nil || flagsErr != nil || index <= 0 || mtu < 0 {
			return nil, errors.New("invalid interface metadata")
		}
		var hardware HardwareAddr
		if fields[3] != "" {
			var err error
			hardware, err = hex.DecodeString(fields[3])
			if err != nil {
				return nil, err
			}
		}
		table = append(table, Interface{Index: index, MTU: mtu, Name: fields[4], HardwareAddr: hardware, Flags: Flags(flags)})
	}
	return table, nil
}

func interfaceAddresses() ([]interfaceAddr, error) {
	_, addrs, cause := dart.Interfaces()
	if cause != "" {
		return nil, errors.New(cause)
	}
	list := make([]interfaceAddr, 0, len(addrs))
	for _, item := range addrs {
		index, text, ok := splitIndex(item)
		if !ok {
			return nil, errors.New("the host gave the address " + item)
		}
		network, err := interfaceIPNet(text)
		if err != nil {
			return nil, err
		}
		list = append(list, interfaceAddr{index: index, addr: network})
	}
	return list, nil
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
		return nil, errors.New("the address " + text + " holds no prefix")
	}
	host, prefix := text[:i], text[i+1:]
	if z := strings.IndexByte(host, '%'); z >= 0 {
		host = host[:z]
	}
	ip := ParseIP(host)
	if ip == nil {
		return nil, errors.New("the address " + text + " holds no IP")
	}
	ones, err := strconv.Atoi(prefix)
	if err != nil {
		return nil, errors.New("the address " + text + " holds no prefix")
	}
	bits := 8 * IPv6len
	if ip.To4() != nil {
		bits = 8 * IPv4len
	}
	mask := CIDRMask(ones, bits)
	if mask == nil {
		return nil, errors.New("the address " + text + " holds no prefix")
	}
	return &IPNet{IP: ip, Mask: mask}, nil
}
