package main

func main() {
	var i8 int8 = 127
	i8++
	println(i8)

	var u8 uint8 = 0
	u8--
	println(u8)

	var i32 int32 = 2147483647
	i32 = i32 + 1
	println(i32)

	var u32 uint32 = 4294967295
	println(u32, u32+1)

	var u64 uint64 = 18446744073709551615
	println(u64 / 3)
	println(u64 % 10)
	println(u64 > 1)

	println(-7/2, -7%2, 7%-2)
	println(1 << 10)
	var x int = -8
	println(x >> 1)
	var y uint = 8
	println(y >> 1)

	println(5 &^ 3)
	println(^5)

	var f32 float32 = 0.1
	var f64 float64 = 0.1
	println(float64(f32) == f64)
	println(f64 * 2)
	f := 3.9
	println(int(f), float64(3))
	println(min(3, 1, 2), max(3, 1, 2))
}
