package sub

import "testing"

func TestSumBlocks(t *testing.T) {
	values := make(chan int)
	go func() {
		values <- 1
		values <- 2
		close(values)
	}()
	if got := Sum(values); got != 3 {
		t.Errorf("Sum gives %d, and 3 is the total", got)
	}
}

func TestBroken(t *testing.T) {
	if got := Broken(); got != 2 {
		t.Errorf("Broken gives %d, and 2 is the result", got)
	}
}

func TestFatal(t *testing.T) {
	defer t.Log("the deferred call ran")
	t.Fatalf("the test stops here")
}

func BenchmarkBroken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Broken()
	}
}
