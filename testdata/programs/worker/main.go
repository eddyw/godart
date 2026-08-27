package main

import "fmt"

// double is a pure function of its input, thus it fits the worker contract.
func double(in []byte) []byte {
	out := make([]byte, len(in))
	for i, b := range in {
		out[i] = b * 2
	}
	return out
}

func sumBytes(in []byte) []byte {
	total := 0
	for _, b := range in {
		total += int(b)
	}
	return []byte{byte(total % 256)}
}

func main() {
	result := runWorker(double, []byte{1, 2, 3})
	fmt.Println(len(result), result[0], result[1], result[2])

	total := runWorker(sumBytes, []byte{10, 20, 30})
	fmt.Println(total[0])

	empty := runWorker(double, []byte{})
	fmt.Println(len(empty))
}
