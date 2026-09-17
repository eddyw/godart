package workload

import (
	"example.com/kernelworkload/settings"
	"math/bits"
)

func Digest(seed uint64, count int) uint64 {
	value := seed
	for i := 0; i < count; i++ {
		value = bits.RotateLeft64(value, i-settings.Bias)
		value ^= bits.Reverse64(uint64(i))
		value += uint64(bits.OnesCount64(value))
		value = bits.ReverseBytes64(value)
	}
	return value
}
