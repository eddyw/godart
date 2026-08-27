package main

func safeIndex(s []int, i int) int {
	return s[i]
}

func main() {
	defer println("deferred runs")
	s := []int{1, 2, 3}
	println(safeIndex(s, 1))
	println(safeIndex(s, 5))
	println("not reached")
}
