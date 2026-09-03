// The program writes the panic text of an index and of a slice bound of an
// unsigned type. A value above 2^63 reads as a negative Dart int, and Go
// writes the unsigned value.
package main

import "fmt"

func report(name string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%s %v\n", name, r)
			return
		}
		fmt.Printf("%s no panic\n", name)
	}()
	f()
}

func main() {
	s := []int{1, 2, 3}
	a := [3]int{1, 2, 3}
	t := "123"
	big := uint64(1<<64 - 1)
	mid := uint64(1<<32 - 1)
	neg := -1

	report("slice[big]", func() { _ = s[big] })
	report("slice[mid]", func() { _ = s[mid] })
	report("slice[neg]", func() { _ = s[neg] })
	report("array[big]", func() { _ = a[big] })
	report("string[big]", func() { _ = t[big] })

	report("write[big]", func() { s[big] = 0 })
	report("writearray[big]", func() { a[big] = 0 })

	report("slice[:big]", func() { _ = s[:big] })
	report("slice[big:]", func() { _ = s[big:] })
	report("slice[:big:3]", func() { _ = s[:big:3] })
	report("slice[:1:big]", func() { _ = s[:1:big] })
	report("string[:big]", func() { _ = t[:big] })
	report("array[:big]", func() { _ = a[:big] })

	report("slice[:neg]", func() { _ = s[:neg] })
	report("slice[neg:]", func() { _ = s[neg:] })
}
