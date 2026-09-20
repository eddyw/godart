package main

type res struct {
	name string
}

func (r res) close() {
	println("close", r.name)
}

func order() {
	for i := 0; i < 3; i++ {
		defer println("defer", i)
	}
	println("body done")
}

func named() (n int) {
	defer func() { n *= 2 }()
	n = 21
	return n
}

func early(flag bool) (text string) {
	defer func() { text = text + "!" }()
	if flag {
		text = "yes"
		return
	}
	return "no"
}

func withRes() {
	r := res{"file"}
	defer r.close()
	r.name = "changed"
	println("work", r.name)
}

func gotoSum(n int) int {
	total := 0
	i := 0
loop:
	if i >= n {
		goto done
	}
	total += i
	i++
	goto loop
done:
	return total
}

func gotoSkip() int {
	x := 1
	goto end
	x = 2
end:
	return x
}

// gotoNested holds a label inside the body of a loop. The flatten pass of
// stage 4 gives such a label its own state.
func gotoNested(n int) int {
	for i := 0; i < n; i++ {
		if i == 3 {
			goto found
		}
		continue
	found:
		return i
	}
	return -1
}

func pair() (int, int) {
	return 3, 4
}

func namedPair() (left, right int) {
	defer func() { right += 10 }()
	return pair()
}

func main() {
	println(namedPair())
	order()
	println(named())
	println(early(true), early(false))
	withRes()
	println(gotoSum(5))
	println(gotoSkip())
	println(gotoNested(6), gotoNested(2))
}
