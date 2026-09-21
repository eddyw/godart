//go:build dart

package time

// minDuration and maxDuration bound a Duration.
const (
	minDuration Duration = -1 << 63
	maxDuration Duration = 1<<63 - 1
)

// Truncate gives d with the part after the last whole multiple of m removed. It
// gives d again for an m that is not positive.
func (d Duration) Truncate(m Duration) Duration {
	if m <= 0 {
		return d
	}
	return d - d%m
}

// Round gives d rounded to the nearest multiple of m. A tie rounds away from
// 0. It gives d again for an m that is not positive.
func (d Duration) Round(m Duration) Duration {
	if m <= 0 {
		return d
	}
	r := d % m
	if d < 0 {
		r = -r
		if lessThanHalf(r, m) {
			return d + r
		}
		if d1 := d - m + r; d1 < d {
			return d1
		}
		return minDuration
	}
	if lessThanHalf(r, m) {
		return d - r
	}
	if d1 := d + m - r; d1 > d {
		return d1
	}
	return maxDuration
}

// lessThanHalf reports whether x+x < y. The test holds against an overflow.
func lessThanHalf(x, y Duration) bool {
	return uint64(x)+uint64(x) < uint64(y)
}

// Truncate gives t rounded down to a multiple of d since January 1 of the year
// 1. It gives t again for a d that is not positive.
func (t Time) Truncate(d Duration) Time {
	t.stripMono()
	if d <= 0 {
		return t
	}
	_, r := div(t, d)
	return t.Add(-r)
}

// Round gives t rounded to the nearest multiple of d since January 1 of the
// year 1. A tie rounds up. It gives t again for a d that is not positive.
func (t Time) Round(d Duration) Time {
	t.stripMono()
	if d <= 0 {
		return t
	}
	_, r := div(t, d)
	if lessThanHalf(r, d) {
		return t.Add(-r)
	}
	return t.Add(d - r)
}

// div gives the parity of the quotient of t by d, and the remainder. The count
// of nanoseconds of t since January 1 of the year 1 does not fit an int64, thus
// the general case builds a 128 bit value from 2 words.
func div(t Time, d Duration) (int, Duration) {
	neg := false
	sec := t.sec()
	nsec := int64(t.nsec())
	if sec < 0 {
		// The 2 counts take their magnitude, and the parity and the remainder
		// read the sign again at the end.
		neg = true
		sec = -sec
		nsec = -nsec
		if nsec < 0 {
			nsec += 1000000000
			sec--
		}
	}
	qmod2 := 0
	var r Duration
	switch {
	case d < Second && Second%(d+d) == 0:
		// 2 times d divides 1 second, thus the seconds add nothing.
		qmod2 = int(nsec/int64(d)) & 1
		r = Duration(nsec % int64(d))
	case d%Second == 0:
		// d holds a whole count of seconds.
		d1 := int64(d / Second)
		qmod2 = int(sec/d1) & 1
		r = Duration(sec%d1)*Second + Duration(nsec)
	default:
		// The count of nanoseconds takes 2 words of 64 bits.
		usec := uint64(sec)
		tmp := (usec >> 32) * 1e9
		u1 := tmp >> 32
		u0 := tmp << 32
		tmp = (usec & 0xFFFFFFFF) * 1e9
		u0x := u0
		u0 += tmp
		if u0 < u0x {
			u1++
		}
		u0x = u0
		u0 += uint64(nsec)
		if u0 < u0x {
			u1++
		}
		// The remainder comes from a subtraction of d shifted left, with a
		// smaller shift on each pass. The last pass gives the parity.
		d1 := uint64(d)
		for d1>>63 != 1 {
			d1 <<= 1
		}
		d0 := uint64(0)
		for {
			qmod2 = 0
			if u1 > d1 || u1 == d1 && u0 >= d0 {
				qmod2 = 1
				u0x = u0
				u0 -= d0
				if u0 > u0x {
					u1--
				}
				u1 -= d1
			}
			if d1 == 0 && d0 == uint64(d) {
				break
			}
			d0 >>= 1
			d0 |= (d1 & 1) << 63
			d1 >>= 1
		}
		r = Duration(u0)
	}
	if neg && r != 0 {
		qmod2 ^= 1
		r = d - r
	}
	return qmod2, r
}

// unitMap gives the count of nanoseconds of each unit that ParseDuration reads.
var unitMap = map[string]uint64{
	"ns": uint64(Nanosecond),
	"us": uint64(Microsecond),
	"µs": uint64(Microsecond), // U+00B5, the micro sign
	"μs": uint64(Microsecond), // U+03BC, the Greek letter mu
	"ms": uint64(Millisecond),
	"s":  uint64(Second),
	"m":  uint64(Minute),
	"h":  uint64(Hour),
}

type parseDurationError struct {
	message string
	text    string
}

func (e *parseDurationError) Error() string {
	return "time: " + e.message + " " + quote(e.text)
}

// ParseDuration reads a duration, as in "300ms", "-1.5h", or "2h45m". The units
// are "ns", "us", "µs", "ms", "s", "m", and "h".
func ParseDuration(s string) (Duration, error) {
	orig := s
	var d uint64
	neg := false

	if s != "" {
		c := s[0]
		if c == '-' || c == '+' {
			neg = c == '-'
			s = s[1:]
		}
	}
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, &parseDurationError{"invalid duration", orig}
	}
	for s != "" {
		var v, f uint64
		scale := float64(1)

		if !(s[0] == '.' || '0' <= s[0] && s[0] <= '9') {
			return 0, &parseDurationError{"invalid duration", orig}
		}
		pl := len(s)
		var ok bool
		v, s, ok = leadingInt(s)
		if !ok {
			return 0, &parseDurationError{"invalid duration", orig}
		}
		pre := pl != len(s)

		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, scale, s = leadingFraction(s)
			post = pl != len(s)
		}
		if !pre && !post {
			return 0, &parseDurationError{"invalid duration", orig}
		}

		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return 0, &parseDurationError{"missing unit in duration", orig}
		}
		u := s[:i]
		s = s[i:]
		unit, known := unitMap[u]
		if !known {
			return 0, &parseDurationError{"unknown unit " + quote(u) + " in duration", orig}
		}
		if v > 1<<63/unit {
			return 0, &parseDurationError{"invalid duration", orig}
		}
		v *= unit
		if f > 0 {
			// A fraction of an hour needs a float64 to stay exact to the
			// nanosecond.
			v += uint64(float64(f) * (float64(unit) / scale))
			if v > 1<<63 {
				return 0, &parseDurationError{"invalid duration", orig}
			}
		}
		d += v
		if d > 1<<63 {
			return 0, &parseDurationError{"invalid duration", orig}
		}
	}
	if neg {
		return -Duration(d), nil
	}
	if d > 1<<63-1 {
		return 0, &parseDurationError{"invalid duration", orig}
	}
	return Duration(d), nil
}

// leadingInt reads the digits at the front of s. It reports false for a value
// that does not fit.
func leadingInt(s string) (uint64, string, bool) {
	x := uint64(0)
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > 1<<63/10 {
			return 0, "", false
		}
		x = x*10 + uint64(c) - '0'
		if x > 1<<63 {
			return 0, "", false
		}
	}
	return x, s[i:], true
}

// leadingFraction reads the digits at the front of s and the scale of them. A
// digit that does not fit reaches no result, and it changes no scale.
func leadingFraction(s string) (uint64, float64, string) {
	x := uint64(0)
	scale := float64(1)
	overflow := false
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if overflow {
			continue
		}
		if x > (1<<63-1)/10 {
			overflow = true
			continue
		}
		y := x*10 + uint64(c) - '0'
		if y > 1<<63 {
			overflow = true
			continue
		}
		x = y
		scale *= 10
	}
	return x, scale, s[i:]
}
