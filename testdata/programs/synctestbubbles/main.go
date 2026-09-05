// Command synctestbubbles proves that 2 bubbles of testing/synctest run at
// the same time.
//
// Each bubble holds a clock of its own, and each one reaches its end. Go
// leaves the order of the 2 open. Each bubble thus keeps its own lines, and
// the program writes them in the order of the names.
package main

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// names holds the bubbles in the order of the output.
var names = []string{"A", "B"}

// steps gives the durations of 1 bubble. The 2 bubbles hold a different set,
// thus each clock reaches a value of its own.
var steps = map[string][]time.Duration{
	"A": {time.Second, 2 * time.Second, 3 * time.Second},
	"B": {5 * time.Second, time.Minute, time.Hour},
}

func run(name string) []string {
	out := []string{}
	t := new(testing.T)
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		out = append(out, "start "+start.UTC().Format(time.RFC3339))
		for i, d := range steps[name] {
			done := make(chan bool)
			go func() {
				time.Sleep(d)
				done <- true
			}()
			<-done
			out = append(out, fmt.Sprint("step ", i, " ", time.Since(start)))
		}
		out = append(out, "end "+time.Now().UTC().Format(time.RFC3339))
	})
	return out
}

func main() {
	// The lock, the map, and the group stand outside each bubble.
	var mu sync.Mutex
	var wg sync.WaitGroup
	lines := map[string][]string{}
	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			out := run(name)
			mu.Lock()
			lines[name] = out
			mu.Unlock()
		}(name)
	}
	wg.Wait()
	for _, name := range names {
		for _, line := range lines[name] {
			fmt.Println(name, line)
		}
	}
}
