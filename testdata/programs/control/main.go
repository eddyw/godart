package main

func classify(n int) string {
	switch {
	case n < 0:
		return "negative"
	case n == 0:
		return "zero"
	}
	return "positive"
}

func main() {
	for i := 0; i < 3; i++ {
		if i == 1 {
			continue
		}
		println(i)
	}

	println(classify(-1), classify(0), classify(5))

	switch n := 2; n {
	case 1:
		println("one")
	case 2:
		println("two")
		fallthrough
	case 3:
		println("three")
	default:
		println("other")
	}

	switch 9 {
	case 1:
		println("one")
	default:
		println("default")
	}

outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue outer
			}
			if i == 2 {
				break outer
			}
			println(i, j)
		}
	}

	n := 0
	for {
		n++
		if n > 2 {
			break
		}
	}
	println(n)

	for i := range 3 {
		println(i)
	}

	k := 0
	for k < 3 {
		k += 2
	}
	println(k)

	switch x := 5; {
	case x > 3:
		println("big")
		break
	}
}
