package main

import "testing"

func TestValue(t *testing.T) {
	if got := value(); got != 7 {
		t.Errorf("value gives %d, and 7 is the number", got)
	}
}
