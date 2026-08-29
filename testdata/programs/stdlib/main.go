package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// lengths orders each word by its length. It holds the 3 operations of
// sort.Interface.
type lengths []string

func (l lengths) Len() int { return len(l) }

func (l lengths) Less(i, j int) bool { return len(l[i]) < len(l[j]) }

func (l lengths) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

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
	fmt.Println(strings.FieldsFunc("a1b22c333", func(r rune) bool { return r >= '0' && r <= '9' }))
	fmt.Println(strings.FieldsFunc("", func(r rune) bool { return true }))
	fmt.Println(strings.FieldsFunc("xx", func(r rune) bool { return true }))
	fmt.Println(strings.FieldsFunc("héllo wörld", func(r rune) bool { return r == ' ' }))
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
	fmt.Println(string(strconv.AppendInt([]byte("n="), -7, 10)))
	fmt.Println(strconv.FormatUint(255, 16), strconv.FormatUint(0, 2))
	fmt.Println(strconv.CanBackquote("plain"), strconv.CanBackquote("a`b"))
	fmt.Println(strconv.CanBackquote("a\tb"), strconv.CanBackquote("a\nb"))
	fmt.Println(strconv.QuoteToASCII("a\tb\u00e9\U0001F600"))
	fmt.Println(strconv.QuoteToASCII("\a\b\f\v\x00\x7f"))

	nums := []int{5, 2, 9, 1}
	sort.Ints(nums)
	fmt.Println(nums)
	words := []string{"pear", "apple", "fig"}
	sort.Strings(words)
	fmt.Println(words)
	sort.Slice(nums, func(i, j int) bool { return nums[i] > nums[j] })
	fmt.Println(nums)
	fmt.Println(sort.SearchInts([]int{1, 3, 5, 7}, 5))
	byLength := lengths{"pear", "fig", "apple"}
	sort.Sort(byLength)
	fmt.Println(byLength)
	sort.Stable(lengths{"bb", "a", "cc"})
	at, found := sort.Find(4, func(i int) int { return 5 - []int{1, 3, 5, 7}[i] })
	fmt.Println(at, found)
	_, missing := sort.Find(4, func(i int) int { return 4 - []int{1, 3, 5, 7}[i] })
	fmt.Println(missing)

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
