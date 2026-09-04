package main

import (
	"fmt"
	"net"
)

func main() {
	// The open joins the group. A machine that carries no route to a
	// multicast group reports the cause here.
	group := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 250), Port: 9251}
	c, err := net.ListenMulticastUDP("udp4", nil, group)
	fmt.Println("open:", err == nil)
	if err == nil {
		fmt.Println("network:", c.LocalAddr().Network())
		a, _ := c.LocalAddr().(*net.UDPAddr)
		fmt.Println("port:", a.Port)
		fmt.Println("close:", c.Close() == nil)
	}

	// A second member of the group shares the port of the first one.
	c1, err1 := net.ListenMulticastUDP("udp4", nil, group)
	c2, err2 := net.ListenMulticastUDP("udp4", nil, group)
	fmt.Println("two members:", err1 == nil && err2 == nil)
	if err1 == nil {
		c1.Close()
	}
	if err2 == nil {
		c2.Close()
	}

	_, err = net.ListenMulticastUDP("tcp", nil, group)
	fmt.Println("bad network:", err != nil)
	_, err = net.ListenMulticastUDP("udp4", nil, nil)
	fmt.Println("no address:", err != nil)
}
