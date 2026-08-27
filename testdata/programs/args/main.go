package main

import (
	"fmt"
	"strings"
)

func main() {
	list := arguments()
	fmt.Println("count", len(list))
	fmt.Println("text", strings.Join(list, ","))
}
