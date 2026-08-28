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

	// A for clause that assigns 2 variables needs a block, because each right
	// side reads the old value.
	s := []int{1, 2, 3, 4, 5}
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	println(s[0], s[4])

	// A continue statement must run the third clause of such a loop.
	sum := 0
	for i, step := 0, 1; i < 10; i, step = i+step, step+1 {
		if i%2 == 0 {
			continue
		}
		sum += i
	}
	println(sum)

	found := 0
wide:
	for i, c := 0, 0; i < 4; i, c = i+1, c+2 {
		for j := 0; j < 4; j++ {
			if i*j == 6 {
				found = c
				break wide
			}
			if j == 2 {
				continue wide
			}
		}
	}
	println(found)

	steps := 0
	for a, b := 1, 2; a+b < 20; {
		steps++
		a, b = b, a+b
	}
	println(steps)
}
