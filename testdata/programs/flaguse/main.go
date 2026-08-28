package main

import (
	"flag"
	"fmt"
	"os"
)

type list struct{ items []string }

func (l *list) String() string { return fmt.Sprint(l.items) }

func (l *list) Set(v string) error {
	l.items = append(l.items, v)
	return nil
}

func main() {
	set := flag.NewFlagSet("example", flag.ContinueOnError)
	set.SetOutput(os.Stdout)
	name := set.String("name", "none", "the `name` of the thing")
	count := set.Int("count", 3, "how many")
	rate := set.Float64("rate", 1.5, "the rate")
	quiet := set.Bool("quiet", false, "write no line")
	size := set.Int64("size", 0, "the size in bytes")
	total := set.Uint("total", 7, "the total")
	var items list
	set.Var(&items, "item", "an item, once for each value")

	args := []string{
		"-name=gart", "-count", "5", "--rate=2.25", "-quiet",
		"-size", "0x10", "-item", "a", "-item", "b", "rest1", "rest2",
	}
	if err := set.Parse(args); err != nil {
		fmt.Println("parse:", err)
	}
	fmt.Println("name:", *name)
	fmt.Println("count:", *count)
	fmt.Println("rate:", *rate)
	fmt.Println("quiet:", *quiet)
	fmt.Println("size:", *size)
	fmt.Println("total:", *total)
	fmt.Println("items:", items.String())
	fmt.Println("args:", set.Args(), set.NArg(), set.Arg(0), set.Arg(9))
	fmt.Println("flags:", set.NFlag(), set.Parsed())
	fmt.Println("lookup:", set.Lookup("name").Name, set.Lookup("name").DefValue)

	set.VisitAll(func(f *flag.Flag) { fmt.Println("all:", f.Name, f.DefValue) })
	set.Visit(func(f *flag.Flag) { fmt.Println("set:", f.Name, f.Value.String()) })

	fmt.Println("defaults:")
	set.PrintDefaults()

	second := flag.NewFlagSet("second", flag.ContinueOnError)
	second.SetOutput(os.Stdout)
	second.Bool("on", true, "a switch")
	if err := second.Parse([]string{"-on=false", "--", "-notaflag"}); err != nil {
		fmt.Println("second:", err)
	}
	fmt.Println("second args:", second.Args())

	third := flag.NewFlagSet("third", flag.ContinueOnError)
	third.SetOutput(os.Stdout)
	if err := third.Parse([]string{"-missing"}); err != nil {
		fmt.Println("third:", err)
	}
}
