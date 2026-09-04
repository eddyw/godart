package envuse

import (
	"bytes"
	"crypto/rand"
	"testing"
	"testing/cryptotest"
)

// TestGlobalRandom proves that a seed of cryptotest gives the same bytes each
// time, and that another seed gives other bytes.
func TestGlobalRandom(t *testing.T) {
	read := func(n int) []byte {
		out := make([]byte, n)
		if _, err := rand.Read(out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	cryptotest.SetGlobalRandom(t, 7)
	first := read(16)

	cryptotest.SetGlobalRandom(t, 7)
	second := read(16)
	if !bytes.Equal(first, second) {
		t.Errorf("2 reads of the seed 7 differ: %x and %x", first, second)
	}

	cryptotest.SetGlobalRandom(t, 8)
	third := read(16)
	if bytes.Equal(first, third) {
		t.Errorf("the seed 8 gives the bytes of the seed 7: %x", third)
	}
}

// TestGlobalRandomAfterCleanup proves that the cleanup of the test above put
// the source of crypto/rand back.
func TestGlobalRandomAfterCleanup(t *testing.T) {
	out := make([]byte, 16)
	if _, err := rand.Read(out); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(out, make([]byte, 16)) {
		t.Error("the source of crypto/rand gives 16 zero bytes")
	}
}

// TestGlobalRandomRejectsParallel proves that a parallel test reaches no
// global source. Such a test runs at the same time as another test.
func TestGlobalRandomRejectsParallel(t *testing.T) {
	t.Run("parallel", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Error("SetGlobalRandom accepted a parallel test")
			}
		}()
		cryptotest.SetGlobalRandom(t, 1)
	})
}
