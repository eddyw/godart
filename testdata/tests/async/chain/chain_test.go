package chain

import "testing"

func TestChainsAgree(t *testing.T) {
	if got, want := SyncChain(Depth), AsyncChain(Depth); got != want {
		t.Errorf("SyncChain gives %d, and AsyncChain gives %d", got, want)
	}
}

func BenchmarkSyncChain(b *testing.B) {
	total := 0
	for i := 0; i < b.N; i++ {
		total += SyncChain(Depth)
	}
	Sink = total
}

func BenchmarkAsyncChain(b *testing.B) {
	total := 0
	for i := 0; i < b.N; i++ {
		total += AsyncChain(Depth)
	}
	Sink = total
}
