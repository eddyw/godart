package main

import "fmt"

func main() {
	vals := []int{0, 5, 47, 255, 4096, -5, -255}
	verbs := []string{
		"%#x", "%#X", "%#o", "%#O", "%#b",
		"%#04x", "%#04X", "%#04o", "%#04O", "%#04b",
		"%#8x", "%#-8x", "%08x", "%08d", "%+08d", "%#010x",
		"%04d", "%+04d", "% 04d", "%#4o", "%#2x",
		"%O", "%04O", "%o", "%b", "%x",
	}
	for _, v := range vals {
		for _, verb := range verbs {
			fmt.Printf("%s of %d = [%s]\n", verb, v, fmt.Sprintf(verb, v))
		}
	}
	fmt.Printf("[%#04x] [%#04x] [%#o]\n", uint8(7), uint16(4095), uint8(8))
}
