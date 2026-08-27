package main

func sum(values []int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}

func main() {
	s := make([]int, 3, 8)
	println(len(s), cap(s))
	s[0] = 1
	s[1] = 2
	s[2] = 3
	println(sum(s))

	s = append(s, 4, 5)
	println(len(s), cap(s), sum(s))

	t := s[1:3]
	println(len(t), cap(t), t[0])
	t[0] = 20
	println(s[1])

	u := []int{9, 8, 7}
	println(len(u), u[2])

	dst := make([]int, 2)
	n := copy(dst, u)
	println(n, dst[0], dst[1])

	var nilSlice []int
	println(len(nilSlice), nilSlice == nil)
	nilSlice = append(nilSlice, 1)
	println(len(nilSlice), nilSlice[0])

	grid := [][]int{{1, 2}, {3, 4}}
	println(grid[1][0])

	for i, v := range u {
		println(i, v)
	}

	bytes := []byte{65, 66}
	println(len(bytes), bytes[0])
}
