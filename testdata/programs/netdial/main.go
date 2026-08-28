package main

import "fmt"

// Port 0 accepts no connection. Both runtimes report a failure of the dial,
// and each one writes a different text, thus the program writes only the flag.
const port = 0

func main() {
	fmt.Println("dial failed:", dialFailed(port))
	fmt.Println("second dial failed:", dialFailed(port))
}
