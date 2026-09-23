package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile("/etc/protocols")
	if os.IsNotExist(err) {
		fmt.Println("protocol database unavailable")
		return
	}
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line, _, _ = strings.Cut(line, "#")
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for _, name := range append(fields[:1:1], fields[2:]...) {
			for _, protocol := range []string{name, strings.ToLower(name), strings.ToUpper(name)} {
				addr, err := net.ResolveIPAddr("ip:"+protocol, "127.0.0.1")
				fmt.Printf("%q: %v %T %v\n", protocol, addr, err, err)
			}
		}
	}
}
