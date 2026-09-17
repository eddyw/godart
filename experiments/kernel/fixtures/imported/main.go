package main

import (
	"example.com/kernelworkload/workload"
	"math/bits"
)

func sample(value uint64) {
	println(int(value), ^value, -value)
	println(value < 1<<63, value <= 1<<63, value > 1<<63, value >= 1<<63)
	println(value<<64, value>>64, value>>uint64(1<<64-1))
	println(value, bits.OnesCount64(value), bits.Reverse64(value), bits.ReverseBytes64(value))
	for shift := -65; shift <= 65; shift++ {
		println(bits.RotateLeft64(value, shift))
	}
	println(workload.Digest(value, 128))
}

func main() {
	sample(0)
	sample(1)
	sample(1 << 63)
	sample(1<<64 - 1)
	sample(0x0123456789abcdef)
	value := uint64(17)
	for i := 0; i < 256; i++ {
		value = value*6364136223846793005 + 1442695040888963407
		println(workload.Digest(value, 32))
	}
}
