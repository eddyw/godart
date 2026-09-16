package main

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"
)

func main() {
	synctest.Test(new(testing.T), func(t *testing.T) {
		start := time.Now()
		for _, delay := range []time.Duration{1, 127, 129, 1001, 999999999} {
			time.Sleep(delay)
			now := time.Now()
			fmt.Println(now.UnixNano(), now.Sub(start))
		}
	})
}
