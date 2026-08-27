package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	s := "Hello, World"
	fmt.Println(strings.ToUpper(s), strings.ToLower(s))
	fmt.Println(strings.Contains(s, "World"), strings.HasPrefix(s, "He"), strings.HasSuffix(s, "ld"))
	fmt.Println(strings.Index(s, "o"), strings.LastIndex(s, "o"))
	fmt.Println(strings.Split("a,b,c", ","))
	fmt.Println(strings.Join([]string{"x", "y"}, "-"))
	fmt.Println(strings.ReplaceAll("aaa", "a", "b"))
	fmt.Println("[" + strings.TrimSpace("  pad  ") + "]")
	fmt.Println(strings.Repeat("ab", 3), strings.Count("cheese", "e"))
	fmt.Println(strings.Fields(" a  b c "))
	before, after, found := strings.Cut("key=value", "=")
	fmt.Println(before, after, found)

	var b strings.Builder
	b.WriteString("one")
	b.WriteString("-two")
	fmt.Println(b.String(), b.Len())

	fmt.Println(strconv.Itoa(-42), strconv.FormatInt(255, 16))
	n, err := strconv.Atoi("123")
	fmt.Println(n, err)
	_, err = strconv.Atoi("x1")
	fmt.Println(err)
	f, _ := strconv.ParseFloat("2.5", 64)
	fmt.Println(f, strconv.FormatFloat(3.14159, 'f', 2, 64))
	fmt.Println(strconv.FormatBool(true), strconv.Quote("a\tb"))

	nums := []int{5, 2, 9, 1}
	sort.Ints(nums)
	fmt.Println(nums)
	words := []string{"pear", "apple", "fig"}
	sort.Strings(words)
	fmt.Println(words)
	sort.Slice(nums, func(i, j int) bool { return nums[i] > nums[j] })
	fmt.Println(nums)
	fmt.Println(sort.SearchInts([]int{1, 3, 5, 7}, 5))

	var mu sync.Mutex
	var wg sync.WaitGroup
	total := 0
	for i := 1; i <= 4; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			mu.Lock()
			total += v
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	fmt.Println("total", total)

	var once sync.Once
	count := 0
	for range 3 {
		once.Do(func() { count++ })
	}
	fmt.Println("once", count)

	start := time.Now()
	time.Sleep(20 * time.Millisecond)
	if time.Since(start) >= 10*time.Millisecond {
		fmt.Println("slept")
	}
	fmt.Println(time.Duration(1500) * time.Millisecond)
	fmt.Println((2500 * time.Microsecond).String())
}
