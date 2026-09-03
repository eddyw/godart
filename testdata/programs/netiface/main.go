// Command netiface reads the network interface table of the host.
//
// A name of an interface and an address of an interface each change between
// 2 machines. Thus each line below names a property alone.
package main

import (
	"fmt"
	"net"
)

// shape reports whether each interface holds a positive index and a name.
func shape(list []net.Interface) bool {
	for _, ifi := range list {
		if ifi.Index <= 0 || ifi.Name == "" {
			return false
		}
	}
	return true
}

// table reads the whole table and runs the 2 lookups.
func table() {
	list, err := net.Interfaces()
	fmt.Println("interfaces:", err == nil)
	if err != nil {
		return
	}
	fmt.Println("at least 1:", len(list) > 0)
	fmt.Println("shape:", shape(list))
	if len(list) == 0 {
		return
	}

	first := list[0]
	byName, err := net.InterfaceByName(first.Name)
	fmt.Println("by name:", err == nil && byName != nil &&
		byName.Name == first.Name && byName.Index == first.Index)
	byIndex, err := net.InterfaceByIndex(first.Index)
	fmt.Println("by index:", err == nil && byIndex != nil &&
		byIndex.Index == first.Index)
}

// lookupErrors runs the 4 rejections of the 2 lookups.
func lookupErrors() {
	_, err := net.InterfaceByName("")
	fmt.Println("empty name:", err)
	_, err = net.InterfaceByName("gart-no-such-interface")
	fmt.Println("unknown name:", err)
	_, err = net.InterfaceByIndex(0)
	fmt.Println("index 0:", err)
	_, err = net.InterfaceByIndex(-1)
	fmt.Println("index -1:", err)
	_, err = net.InterfaceByIndex(1 << 20)
	fmt.Println("unknown index:", err)

	var none *net.Interface
	_, err = none.Addrs()
	fmt.Println("nil interface:", err)
}

// addresses proves the shape of each address of each interface.
func addresses() {
	list, err := net.Interfaces()
	if err != nil {
		fmt.Println("interfaces:", err)
		return
	}
	count := 0
	good := true
	for i := range list {
		found, err := list[i].Addrs()
		if err != nil {
			good = false
			continue
		}
		for _, a := range found {
			count++
			network, ok := a.(*net.IPNet)
			if !ok || len(network.IP) == 0 || len(network.Mask) == 0 {
				good = false
				continue
			}
			if a.Network() != "ip+net" {
				good = false
			}
			ones, bits := network.Mask.Size()
			if bits == 0 || ones > bits {
				good = false
			}
		}
	}
	fmt.Println("addrs shape:", good)
	fmt.Println("addrs count:", count > 0)

	all, err := net.InterfaceAddrs()
	fmt.Println("interface addrs:", err == nil, len(all) > 0)
	fmt.Println("addrs agree:", len(all) == count)

	// The loopback address stands in the table of each host.
	loopback := false
	for _, a := range all {
		if network, ok := a.(*net.IPNet); ok && network.IP.IsLoopback() {
			loopback = true
		}
	}
	fmt.Println("loopback:", loopback)
}

func main() {
	table()
	lookupErrors()
	addresses()
}
