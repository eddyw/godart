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

func main() {
	order()
	println(named())
	println(early(true), early(false))
	withRes()
	println(gotoSum(5))
	println(gotoSkip())
}
