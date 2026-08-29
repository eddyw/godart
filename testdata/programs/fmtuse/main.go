package main

import (
	"errors"
	"fmt"
)

type point struct {
	X int
	Y string
}

type temp float64

func (t temp) String() string {
	return fmt.Sprintf("%.1f degrees", float64(t))
}

func main() {
	fmt.Println("hello", 42, true)
	fmt.Print("a", "b", 1, 2, "\n")
	fmt.Printf("%d %s %v %t\n", 7, "text", 1.5, false)
	fmt.Printf("%q %v %%\n", "quoted", []int{1, 2, 3})

	p := point{1, "two"}
	fmt.Printf("%v %+v\n", p, p)
	fmt.Println(&p)

	m := map[string]int{"b": 2, "a": 1}
	fmt.Printf("%v\n", m)

	fmt.Printf("[%5d] [%-5d] [%5s]\n", 42, 42, "ab")
	fmt.Printf("[%05d] [%05d] [%-05d] [%05s] [%08.2f] [%08.2f] [%05x] [%+06d]\n",
		42, -42, 42, "ab", -1.5, 1.5, 255, 42)
	fmt.Printf("[%+d] [%+d] [%+f] [%+.2f] [%+x] [%+o] [%+b] [%+e] [%+g] [%+s] [%+q] [%+c] [%+v] [%+t]\n",
		42, -42, 1.5, -1.5, 255, 8, 5, 1500.0, 1.5, "ab", "ab", 65, 42, true)
	fmt.Printf("[%+d] [%+x]\n", uint(7), uint(7))
	fmt.Printf("%.3f %e\n", 3.14159, 1234.5)
	fmt.Printf("%T %T %T\n", p, m, 1.5)

	err := errors.New("bad thing")
	fmt.Println(err)
	wrapped := fmt.Errorf("context: %v", err)
	fmt.Println(wrapped)
	fmt.Println(errors.Is(err, err))

	fmt.Println(temp(21.55))
	fmt.Println(fmt.Sprint("x", 1, 2, "y"))
	fmt.Println(fmt.Sprintln(1, "a"))

	var nilErr error
	fmt.Println(nilErr == nil)
	fmt.Printf("%v\n", nilErr)
}
