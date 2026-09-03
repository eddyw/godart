package seed

import "testing"

func TestReverse(t *testing.T) {
	if got := string(Reverse([]byte("abc"))); got != "cba" {
		t.Errorf("Reverse gives %q, and %q is the value", got, "cba")
	}
}

func FuzzBytes(f *testing.F) {
	f.Add([]byte("ab"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		if got := Reverse(Reverse(data)); string(got) != string(data) {
			t.Errorf("2 reverses give %q, and %q is the value", got, data)
		}
	})
}

func FuzzText(f *testing.F) {
	f.Add("ab", 2)
	f.Add("", 3)
	f.Fuzz(func(t *testing.T, text string, n int) {
		if got := Repeat(text, n); n > 0 && len(got) != len(text)*n {
			t.Errorf("Repeat gives %d bytes, and %d is the count", len(got), len(text)*n)
		}
		t.Logf("text %q count %d", text, n)
	})
}

func FuzzSigned(f *testing.F) {
	f.Add(int8(-3), int16(300), int32(-70000), int64(-5000000000))
	f.Add(int8(0), int16(0), int32(0), int64(0))
	f.Fuzz(func(t *testing.T, a int8, b int16, c int32, d int64) {
		t.Logf("%d %d %d %d", a, b, c, d)
	})
}

func FuzzUnsigned(f *testing.F) {
	f.Add(uint(7), uint8(200), uint16(60000), uint32(4000000000), uint64(18000000000000000000))
	f.Add(uint(0), uint8(0), uint16(0), uint32(0), uint64(0))
	f.Fuzz(func(t *testing.T, a uint, b uint8, c uint16, d uint32, e uint64) {
		t.Logf("%d %d %d %d %d", a, b, c, d, e)
	})
}

func FuzzOther(f *testing.F) {
	f.Add(true, byte('a'), 'x', float32(0.5), 1.5)
	f.Add(false, byte(0), '\n', float32(-0.25), -2.5)
	f.Fuzz(func(t *testing.T, flag bool, b byte, r rune, g float32, h float64) {
		t.Logf("%v %d %q %v %v", flag, b, r, g, h)
	})
}

func FuzzReport(f *testing.F) {
	f.Add(1)
	f.Fuzz(func(t *testing.T, n int) {
		if n < 0 {
			t.Errorf("the count %d stands below 0", n)
		}
	})
}
