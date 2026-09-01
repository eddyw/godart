package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	sortRun()
	stringsRun()
	timeRun()
	flagRun()
	syncRun()
	osRun()
	debugRun()
}

func sortRun() {
	x := sort.IntSlice{3, 1, 2}
	x.Sort()
	fmt.Println(x, x.Search(2), sort.IsSorted(x))
	sort.Sort(sort.Reverse(x))
	fmt.Println(x, sort.IsSorted(x))

	s := sort.StringSlice{"c", "a", "b"}
	s.Sort()
	fmt.Println(s, s.Search("b"), sort.SearchStrings(s, "z"))

	f := sort.Float64Slice{2.5, 1.5}
	f.Sort()
	fmt.Println(f, f.Search(2.5), sort.Float64sAreSorted(f))

	fmt.Println(sort.StringsAreSorted([]string{"a", "b"}),
		sort.StringsAreSorted([]string{"b", "a"}),
		sort.SliceIsSorted([]int{1, 2}, func(i, j int) bool { return i < j }))
}

func stringsRun() {
	fmt.Println(strings.SplitAfter("a,b,c", ","))
	fmt.Println(strings.SplitAfterN("a,b,c", ",", 2))
	fmt.Println(strings.SplitAfter("abc", ""))
	fmt.Printf("%q\n", strings.Clone("held"))
	fmt.Println(strings.LastIndexAny("go gopher", "go"),
		strings.LastIndexFunc("ab1", func(r rune) bool { return r == 'b' }),
		strings.ContainsFunc("abc", func(r rune) bool { return r == 'c' }))

	before, after, ok := strings.CutLast("a.b.c", ".")
	fmt.Println(before, after, ok)

	fmt.Printf("%q %q\n", strings.ToValidUTF8("a\xffb", "?"), strings.Title("her royal highness"))
	fmt.Printf("%q\n", strings.ToTitle("loud noises"))
	fmt.Printf("%q\n", strings.TrimFunc("123abc456", func(r rune) bool { return r >= '0' && r <= '9' }))

	var lines []string
	for line := range strings.Lines("a\nb\nc") {
		lines = append(lines, line)
	}
	fmt.Printf("%q\n", lines)

	var parts []string
	for part := range strings.SplitAfterSeq("a,b", ",") {
		parts = append(parts, part)
	}
	fmt.Printf("%q\n", parts)

	var fields []string
	for field := range strings.FieldsFuncSeq("a1b2c", func(r rune) bool { return r >= '0' && r <= '9' }) {
		fields = append(fields, field)
	}
	fmt.Printf("%q\n", fields)

	r := strings.NewReplacer("a", "1", "b", "2")
	fmt.Println(r.Replace("abcab"))
	var out bytes.Buffer
	n, err := r.WriteString(&out, "ab")
	fmt.Println(out.String(), n, err == nil)

	var b strings.Builder
	b.WriteString("xy")
	fmt.Println(b.Cap() >= 2)

	reader := strings.NewReader("hello")
	c, _ := reader.ReadByte()
	reader.UnreadByte()
	rune1, _, _ := reader.ReadRune()
	reader.UnreadRune()
	var sink bytes.Buffer
	written, err := reader.WriteTo(&sink)
	fmt.Println(string(c), string(rune1), written, sink.String(), err == nil)
}

func timeRun() {
	t := time.Date(2024, time.February, 29, 12, 30, 45, 500000000, time.UTC)
	fmt.Println(t.UnixMilli(), t.UnixMicro())
	fmt.Println(time.UnixMilli(t.UnixMilli()).UTC().Format(time.RFC3339Nano))
	fmt.Println(time.UnixMicro(t.UnixMicro()).UTC().Format(time.RFC3339Nano))
	fmt.Println(t.AddDate(1, 1, 1).UTC().Format(time.RFC3339))
	fmt.Println(t.AddDate(0, 0, -60).UTC().Format(time.RFC3339))

	name, offset := t.Zone()
	fmt.Println(name, offset, t.IsDST())

	year, week := t.ISOWeek()
	fmt.Println(year, week)
	for _, day := range []time.Time{
		time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2021, time.January, 4, 0, 0, 0, 0, time.UTC),
		time.Date(2019, time.December, 30, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
	} {
		y, w := day.ISOWeek()
		fmt.Print(y, "-", w, " ")
	}
	fmt.Println()

	d := 90 * time.Minute
	fmt.Println(d.Hours(), d.Minutes(), (-d).Abs())
	fmt.Println(t.Compare(t.Add(time.Second)), t.Compare(t), t.Add(time.Second).Compare(t))
	fmt.Println(string(t.AppendFormat(nil, time.RFC3339)))
	fmt.Println(t.GoString())

	raw, err := t.GobEncode()
	fmt.Println(len(raw), err == nil)
	var back time.Time
	fmt.Println(back.GobDecode(raw) == nil, back.Equal(t))
}

func flagRun() {
	set := flag.NewFlagSet("", flag.ContinueOnError)
	set.Init("demo", flag.ContinueOnError)
	set.SetOutput(new(bytes.Buffer))
	wait := set.Duration("wait", time.Second, "the `delay` of the run")
	var seen []string
	set.Func("tag", "a tag", func(v string) error {
		seen = append(seen, v)
		return nil
	})
	count := 0
	set.BoolFunc("loud", "raise the level", func(string) error {
		count++
		return nil
	})
	var stamp time.Time
	set.TextVar(&stamp, "at", time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC), "a time")

	err := set.Parse([]string{"-wait", "90s", "-tag", "a", "-tag", "b", "-loud",
		"-at", "2024-02-29T00:00:00Z", "rest"})
	fmt.Println(*wait, seen, count, stamp.UTC().Format(time.RFC3339), set.Args(), err == nil)

	name, usage := flag.UnquoteUsage(set.Lookup("wait"))
	fmt.Println(name, usage)
	name, _ = flag.UnquoteUsage(set.Lookup("loud"))
	fmt.Printf("%q\n", name)
}

func syncRun() {
	var m sync.Map
	m.Store("a", 1)
	m.Store("b", 2)
	v, ok := m.Load("a")
	fmt.Println(v, ok)
	actual, loaded := m.LoadOrStore("a", 9)
	fmt.Println(actual, loaded)
	actual, loaded = m.LoadOrStore("c", 3)
	fmt.Println(actual, loaded)
	old, swapped := m.Swap("a", 5)
	fmt.Println(old, swapped, m.CompareAndSwap("a", 5, 6), m.CompareAndSwap("a", 5, 7))
	value, deleted := m.LoadAndDelete("b")
	fmt.Println(value, deleted, m.CompareAndDelete("c", 3))
	count := 0
	m.Range(func(key, value any) bool {
		count++
		return true
	})
	fmt.Println(count)
	m.Clear()
	_, ok = m.Load("a")
	fmt.Println(ok)

	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	ready := false
	var wg sync.WaitGroup
	wg.Go(func() {
		mu.Lock()
		for !ready {
			cond.Wait()
		}
		mu.Unlock()
		fmt.Println("woke")
	})
	mu.Lock()
	ready = true
	cond.Broadcast()
	mu.Unlock()
	wg.Wait()

	var rw sync.RWMutex
	fmt.Println(rw.TryLock())
	rw.Unlock()
	fmt.Println(rw.TryRLock())
	rw.RUnlock()
	locker := rw.RLocker()
	locker.Lock()
	locker.Unlock()
	fmt.Println("locker ok")
}

func osRun() {
	fmt.Println(os.Getpid() > 0, os.Getpagesize() > 0, os.Getppid() >= 0)
	host, err := os.Hostname()
	fmt.Println(len(host) > 0, err == nil)
	path, err := os.Executable()
	fmt.Println(len(path) > 0, err == nil)
	home, err := os.UserHomeDir()
	fmt.Println(len(home) > 0, err == nil)

	os.Setenv("STDGAPS", "value")
	fmt.Println(os.ExpandEnv("a ${STDGAPS} b $STDGAPS c $NOPE d"))
	fmt.Println(os.Expand("x $a y", func(string) string { return "1" }))
	os.Unsetenv("STDGAPS")

	fmt.Println(os.IsExist(os.ErrExist), os.IsPermission(os.ErrPermission),
		os.IsNotExist(os.ErrNotExist), os.IsTimeout(os.ErrNotExist))
	fmt.Println(os.IsPathSeparator('/'), os.PathSeparator, os.PathListSeparator, os.DevNull)
	fmt.Println(os.SEEK_SET, os.SEEK_CUR, os.SEEK_END)

	wrapped := os.NewSyscallError("read", os.ErrNotExist)
	fmt.Println(wrapped, errors.Is(wrapped, os.ErrNotExist), os.NewSyscallError("x", nil) == nil)
	fmt.Println(errors.Is(errors.ErrUnsupported, errors.ErrUnsupported), errors.ErrUnsupported)

	info, err := os.Stat(".")
	fmt.Println(err == nil, os.SameFile(info, info), os.SameFile(info, nil))
}

func debugRun() {
	trace := debug.Stack()
	fmt.Println(strings.HasPrefix(string(trace), "goroutine "), len(trace) > 20)
	fmt.Println(debug.SetGCPercent(50), debug.SetGCPercent(100))
	fmt.Println(debug.SetMaxThreads(20000) > 0, debug.SetPanicOnFault(true), debug.SetPanicOnFault(false))
	fmt.Println(debug.SetMemoryLimit(-1) > 0)
	debug.SetTraceback("all")
	debug.FreeOSMemory()
	var stats debug.GCStats
	debug.ReadGCStats(&stats)
	fmt.Println(stats.NumGC >= 0)
	fmt.Println(debug.SetCrashOutput(nil, debug.CrashOptions{}) == nil)
}
