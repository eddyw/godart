package main

import "fmt"

func main() {
	ch := make(chan int, 1)

	var fs []func() int
	for i := 0; i < 3; i++ {
		ch <- i
		<-ch
		fs = append(fs, func() int { return i })
	}
	for _, f := range fs {
		fmt.Println(f())
	}

	var gs []func() string
	for _, s := range []string{"a", "b"} {
		ch <- 0
		<-ch
		gs = append(gs, func() string { return s })
	}
	for _, g := range gs {
		fmt.Println(g())
	}

	var hs []func() int
	for i := range 3 {
		ch <- i
		<-ch
		v := i * 10
		h := func() int { return v }
		v++
		hs = append(hs, h)
	}
	for _, h := range hs {
		fmt.Println(h())
	}

	done := make(chan int, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- i
		}()
	}
	seen := make([]bool, 3)
	for range 3 {
		seen[<-done] = true
	}
	fmt.Println(seen)

	var ps []*int
	for i := 0; i < 3; i++ {
		ch <- i
		<-ch
		ps = append(ps, &i)
	}
	for _, p := range ps {
		fmt.Println(*p)
	}

	var qs []func() int
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			continue
		}
		ch <- i
		<-ch
		qs = append(qs, func() int { return i })
	}
	for _, q := range qs {
		fmt.Println(q())
	}
}
