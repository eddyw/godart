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

	// The flag 0 with the flag -, with a text, and with a precision.
	for _, verb := range []string{"%-08d", "%08d", "%-8d", "% 8d"} {
		fmt.Printf("%s of 5 = [%s]\n", verb, fmt.Sprintf(verb, 5))
	}
	for _, verb := range []string{"%-#08x", "%#x", "%#X", "%08s", "%-8s", "%8s", "%x"} {
		fmt.Printf("%s of hi = [%s]\n", verb, fmt.Sprintf(verb, "hi"))
		fmt.Printf("%s of bytes = [%s]\n", verb, fmt.Sprintf(verb, []byte("hi")))
	}
	for _, verb := range []string{"%08.2f", "%+08.2f", "% 08.2f", "%-08.2f", "%08.3e"} {
		for _, v := range []float64{1.5, -1.5, 0} {
			fmt.Printf("%s of %v = [%s]\n", verb, v, fmt.Sprintf(verb, v))
		}
	}
	for _, verb := range []string{"%+d", "% d", "%+ d", "% +d"} {
		fmt.Printf("%s of 5 = [%s] of -5 = [%s]\n",
			verb, fmt.Sprintf(verb, 5), fmt.Sprintf(verb, -5))
	}
}
