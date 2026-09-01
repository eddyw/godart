package main

import (
	"fmt"
	"sync/atomic"
)

func main() {
	var i32 atomic.Int32
	i32.Store(0b1100)
	fmt.Println(i32.And(0b1010), i32.Load())
	fmt.Println(i32.Or(0b0001), i32.Load())

	var up atomic.Uintptr
	up.Store(7)
	fmt.Println(up.Add(3), up.Swap(2), up.CompareAndSwap(2, 5), up.Load())

	var u uint64 = 0xff
	fmt.Println(atomic.AndUint64(&u, 0x0f), u)
	fmt.Println(atomic.OrUint64(&u, 0xf0), u)
	fmt.Println(atomic.SwapUint32(new(uint32), 4))
	var p uintptr = 9
	fmt.Println(atomic.LoadUintptr(&p), atomic.CompareAndSwapUintptr(&p, 9, 1), p)
}
