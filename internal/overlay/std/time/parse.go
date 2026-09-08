//go:build dart

package time

import "errors"

// errBad marks a chunk of the value that does not fit its chunk of the layout.
// No caller reads it, because Parse builds a ParseError of its own.
var errBad = errors.New("bad value for field")

// ParseError describes a value that does not fit its layout.
type ParseError struct {
	Layout     string
	Value      string
	LayoutElem string
	ValueElem  string
	Message    string
}

// Error gives the text of the error.
func (e *ParseError) Error() string {
	if e.Message == "" {
		return "parsing time " +
			quote(e.Value) + " as " +
			quote(e.Layout) + ": cannot parse " +
			quote(e.ValueElem) + " as " +
			quote(e.LayoutElem)
	}
	return "parsing time " + quote(e.Value) + e.Message
}

const lowerhex = "0123456789abcdef"

// quote wraps the text in 2 quotation marks. A byte outside the printable
// range of ASCII takes the form \xNN.
func quote(s string) string {
	buf := make([]byte, 1, len(s)+2)
	buf[0] = '"'
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 0x80 || c < ' ':
			buf = append(buf, '\\', 'x', lowerhex[c>>4], lowerhex[c&0xF])
		case c == '"' || c == '\\':
			buf = append(buf, '\\', c)
		default:
			buf = append(buf, c)
		}
	}
	return string(append(buf, '"'))
}

// atoi reads a decimal number, with an optional sign.
func atoi(s string) (int, error) {
	neg := false
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if len(s) == 0 {
		return 0, errBad
	}
	x := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || '9' < s[i] {
			return 0, errBad
		}
		x = x*10 + int(s[i]-'0')
	}
	if neg {
		x = -x
	}
	return x, nil
}

// match reports whether the 2 texts of equal length are equal, with no regard
// for the case of a letter.
func match(s1, s2 string) bool {
	for i := 0; i < len(s1); i++ {
		c1 := s1[i]
		c2 := s2[i]
		if c1 != c2 {
			// 'a'-'A' holds 1 bit, thus the or moves a letter to lower case.
			c1 |= 'a' - 'A'
			c2 |= 'a' - 'A'
			if c1 != c2 || c1 < 'a' || c1 > 'z' {
				return false
			}
		}
	}
	return true
}

// lookup finds the first name of the table that stands at the front of the
// value. It gives the index of the name and the rest of the value.
func lookup(tab []string, val string) (int, string, error) {
	for i, v := range tab {
		if len(val) >= len(v) && match(val[:len(v)], v) {
			return i, val[len(v):], nil
		}
	}
	return -1, val, errBad
}

// getnum reads 1 digit or 2 digits. A fixed chunk needs the 2 digits.
func getnum(s string, fixed bool) (int, string, error) {
	if !isDigit(s, 0) {
		return 0, s, errBad
	}
	if !isDigit(s, 1) {
		if fixed {
			return 0, s, errBad
		}
		return int(s[0] - '0'), s[1:], nil
	}
	return int(s[0]-'0')*10 + int(s[1]-'0'), s[2:], nil
}

// getnum3 reads 1 digit, 2 digits, or 3 digits. A fixed chunk needs the 3.
func getnum3(s string, fixed bool) (int, string, error) {
	n, i := 0, 0
	for ; i < 3 && isDigit(s, i); i++ {
		n = n*10 + int(s[i]-'0')
	}
	if i == 0 || fixed && i != 3 {
		return 0, s, errBad
	}
	return n, s[i:], nil
}

func cutspace(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	return s
}

// skip removes the prefix from the value. A run of spaces of the prefix
// matches a run of spaces of the value of another length.
func skip(value, prefix string) (string, error) {
	for len(prefix) > 0 {
		if prefix[0] == ' ' {
			if len(value) > 0 && value[0] != ' ' {
				return value, errBad
			}
			prefix = cutspace(prefix)
			value = cutspace(value)
			continue
		}
		if len(value) == 0 || value[0] != prefix[0] {
			return value, errBad
		}
		prefix = prefix[1:]
		value = value[1:]
	}
	return value, nil
}

func commaOrPeriod(b byte) bool { return b == '.' || b == ',' }

// parseNanoseconds reads a fraction of a second. The first byte holds the
// separator, and nbytes counts the separator and the digits.
func parseNanoseconds(value string, nbytes int) (int, string, error) {
	if !commaOrPeriod(value[0]) {
		return 0, "", errBad
	}
	if nbytes > 10 {
		value = value[:10]
		nbytes = 10
	}
	ns, err := atoi(value[1:nbytes])
	if err != nil {
		return 0, "", err
	}
	if ns < 0 {
		return 0, "fractional second", nil
	}
	// The result holds nanoseconds, thus each missing digit scales it.
	for i := 0; i < 10-nbytes; i++ {
		ns *= 10
	}
	return ns, "", nil
}

// parseSignedOffset gives the length of a signed offset of hours, as in "+03".
// The hour stands in the range 0 to 23. A value outside it gives 0.
func parseSignedOffset(value string) int {
	if value[0] != '-' && value[0] != '+' {
		return 0
	}
	i := 1
	for i < len(value) && isDigit(value, i) {
		i++
	}
	if i == 1 {
		return 0
	}
	x, err := atoi(value[1:i])
	if err != nil || x > 23 {
		return 0
	}
	return i
}

// parseGMT gives the length of a GMT zone. The value starts with "GMT", and an
// offset of hours can follow it.
func parseGMT(value string) int {
	value = value[3:]
	if len(value) == 0 {
		return 3
	}
	return 3 + parseSignedOffset(value)
}

// parseTimeZone gives the length of the name of a zone at the front of the
// value. The name of a zone follows no rule, thus the test reads a run of
// capital letters and a small set of special cases.
func parseTimeZone(value string) (int, bool) {
	if len(value) < 3 {
		return 0, false
	}
	// ChST and MeST are the only zones with a lower case letter.
	if len(value) >= 4 && (value[:4] == "ChST" || value[:4] == "MeST") {
		return 4, true
	}
	if value[:3] == "GMT" {
		return parseGMT(value), true
	}
	// Some zones carry no name, and they take the form +00 instead.
	if value[0] == '+' || value[0] == '-' {
		n := parseSignedOffset(value)
		return n, n > 0
	}
	nUpper := 0
	for ; nUpper < 6; nUpper++ {
		if nUpper >= len(value) {
			break
		}
		if c := value[nUpper]; c < 'A' || 'Z' < c {
			break
		}
	}
	switch nUpper {
	case 5:
		if value[4] == 'T' {
			return 5, true
		}
	case 4:
		if value[3] == 'T' || value[:4] == "WITA" {
			return 4, true
		}
	case 3:
		return 3, true
	}
	return 0, false
}

// isLeap reports whether the year holds 366 days.
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// daysBefore gives the count of days of the year before the first day of the
// month, for a year that is not a leap year.
func daysBefore(m Month) int {
	adj := 0
	if m >= March {
		adj = -2
	}
	return (214*int(m)-211)/7 + adj
}

// daysIn gives the count of days of the month of the year.
func daysIn(m Month, year int) int {
	if m == February {
		if isLeap(year) {
			return 29
		}
		return 28
	}
	return 30 + int((m+m>>3)&1)
}

// Parse reads a time from the text, in the layout. The layout names each part
// through the reference time.
//
// The result holds UTC when the text names no zone. A text that names an offset
// takes the local zone when the local zone holds that offset at the instant,
// and a zone of a fixed offset otherwise.
func Parse(layout, value string) (Time, error) {
	return parse(layout, value, UTC, Local)
}

// ParseInLocation reads a time from the text in the layout, and it reads the
// location for a text that names no zone.
func ParseInLocation(layout, value string, loc *Location) (Time, error) {
	return parse(layout, value, loc, loc)
}

func parse(layout, value string, defaultLocation, local *Location) (Time, error) {
	alayout, avalue := layout, value
	rangeErrString := ""
	amSet := false
	pmSet := false

	year := 0
	month := -1
	day := -1
	yday := -1
	hour := 0
	min := 0
	sec := 0
	nsec := 0
	var z *Location
	zoneOffset := -1
	zoneName := ""

	for {
		var err error
		prefix, std, suffix := nextStdChunk(layout)
		stdstr := layout[len(prefix) : len(layout)-len(suffix)]
		value, err = skip(value, prefix)
		if err != nil {
			return Time{}, newParseError(alayout, avalue, prefix, value, "")
		}
		if std == 0 {
			if len(value) != 0 {
				return Time{}, newParseError(alayout, avalue, "", value,
					": extra text: "+quote(value))
			}
			break
		}
		layout = suffix
		p := ""
		hold := value
		switch std & stdMask {
		case stdYear:
			if len(value) < 2 {
				err = errBad
				break
			}
			p, value = value[0:2], value[2:]
			year, err = atoi(p)
			if err != nil {
				break
			}
			// The Unix epoch starts in December of 1969 in some zones.
			if year >= 69 {
				year += 1900
			} else {
				year += 2000
			}
		case stdLongYear:
			if len(value) < 4 || !isDigit(value, 0) {
				err = errBad
				break
			}
			p, value = value[0:4], value[4:]
			year, err = atoi(p)
		case stdMonth:
			month, value, err = lookup(shortMonthNames, value)
			month++
		case stdLongMonth:
			month, value, err = lookup(longMonthNames, value)
			month++
		case stdNumMonth, stdZeroMonth:
			month, value, err = getnum(value, std == stdZeroMonth)
			if err == nil && (month <= 0 || 12 < month) {
				rangeErrString = "month"
			}
		case stdWeekDay:
			// The day of the week reaches no field. The read holds the test
			// of the text only.
			_, value, err = lookup(shortDayNames, value)
		case stdLongWeekDay:
			_, value, err = lookup(longDayNames, value)
		case stdDay, stdUnderDay, stdZeroDay:
			if std == stdUnderDay && len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
			day, value, err = getnum(value, std == stdZeroDay)
		case stdUnderYearDay, stdZeroYearDay:
			for i := 0; i < 2; i++ {
				if std == stdUnderYearDay && len(value) > 0 && value[0] == ' ' {
					value = value[1:]
				}
			}
			yday, value, err = getnum3(value, std == stdZeroYearDay)
		case stdHour:
			hour, value, err = getnum(value, false)
			if hour < 0 || 24 <= hour {
				rangeErrString = "hour"
			}
		case stdHour12, stdZeroHour12:
			hour, value, err = getnum(value, std == stdZeroHour12)
			if hour < 0 || 12 < hour {
				rangeErrString = "hour"
			}
		case stdMinute, stdZeroMinute:
			min, value, err = getnum(value, std == stdZeroMinute)
			if min < 0 || 60 <= min {
				rangeErrString = "minute"
			}
		case stdSecond, stdZeroSecond:
			sec, value, err = getnum(value, std == stdZeroSecond)
			if err != nil {
				break
			}
			if sec < 0 || 60 <= sec {
				rangeErrString = "second"
				break
			}
			// The value can hold a fraction of a second that the layout does
			// not name. Parse reads it.
			if len(value) >= 2 && commaOrPeriod(value[0]) && isDigit(value, 1) {
				_, next, _ := nextStdChunk(layout)
				next &= stdMask
				if next == stdFracSecond0 || next == stdFracSecond9 {
					break
				}
				n := 2
				for ; n < len(value) && isDigit(value, n); n++ {
				}
				nsec, rangeErrString, err = parseNanoseconds(value, n)
				value = value[n:]
			}
		case stdPM:
			if len(value) < 2 {
				err = errBad
				break
			}
			p, value = value[0:2], value[2:]
			switch p {
			case "PM":
				pmSet = true
			case "AM":
				amSet = true
			default:
				err = errBad
			}
		case stdpm:
			if len(value) < 2 {
				err = errBad
				break
			}
			p, value = value[0:2], value[2:]
			switch p {
			case "pm":
				pmSet = true
			case "am":
				amSet = true
			default:
				err = errBad
			}
		case stdISO8601TZ, stdISO8601ShortTZ, stdISO8601ColonTZ,
			stdISO8601SecondsTZ, stdISO8601ColonSecondsTZ,
			stdNumTZ, stdNumShortTZ, stdNumColonTZ, stdNumSecondsTz,
			stdNumColonSecondsTZ:
			code := std & stdMask
			isISO := code == stdISO8601TZ || code == stdISO8601ShortTZ ||
				code == stdISO8601ColonTZ || code == stdISO8601SecondsTZ ||
				code == stdISO8601ColonSecondsTZ
			if isISO && len(value) >= 1 && value[0] == 'Z' {
				value = value[1:]
				z = UTC
				break
			}
			sign, hh, mm, ss := "", "", "", ""
			switch {
			case code == stdISO8601ColonTZ || code == stdNumColonTZ:
				if len(value) < 6 || value[3] != ':' {
					err = errBad
					break
				}
				sign, hh, mm, ss, value = value[0:1], value[1:3], value[4:6], "00", value[6:]
			case code == stdNumShortTZ || code == stdISO8601ShortTZ:
				if len(value) < 3 {
					err = errBad
					break
				}
				sign, hh, mm, ss, value = value[0:1], value[1:3], "00", "00", value[3:]
			case code == stdISO8601ColonSecondsTZ || code == stdNumColonSecondsTZ:
				if len(value) < 9 || value[3] != ':' || value[6] != ':' {
					err = errBad
					break
				}
				sign, hh, mm, ss, value = value[0:1], value[1:3], value[4:6], value[7:9], value[9:]
			case code == stdISO8601SecondsTZ || code == stdNumSecondsTz:
				if len(value) < 7 {
					err = errBad
					break
				}
				sign, hh, mm, ss, value = value[0:1], value[1:3], value[3:5], value[5:7], value[7:]
			default:
				if len(value) < 5 {
					err = errBad
					break
				}
				sign, hh, mm, ss, value = value[0:1], value[1:3], value[3:5], "00", value[5:]
			}
			if err != nil {
				break
			}
			var hr, mn, sc int
			hr, _, err = getnum(hh, true)
			if err == nil {
				mn, _, err = getnum(mm, true)
				if err == nil {
					sc, _, err = getnum(ss, true)
				}
			}
			// Some texts write an offset of 24 hours, of 60 minutes, or of 60
			// seconds, thus the test reads more than the range of each unit.
			if hr > 24 {
				rangeErrString = "time zone offset hour"
			}
			if mn > 60 {
				rangeErrString = "time zone offset minute"
			}
			if sc > 60 {
				rangeErrString = "time zone offset second"
			}
			zoneOffset = (hr*60+mn)*60 + sc
			switch sign[0] {
			case '+':
			case '-':
				zoneOffset = -zoneOffset
			default:
				err = errBad
			}
		case stdTZ:
			if len(value) >= 3 && value[0:3] == "UTC" {
				z = UTC
				value = value[3:]
				break
			}
			n, ok := parseTimeZone(value)
			if !ok {
				err = errBad
				break
			}
			zoneName, value = value[:n], value[n:]
		case stdFracSecond0:
			// The layout names the exact count of digits.
			ndigit := 1 + digitsLen(std)
			if len(value) < ndigit {
				err = errBad
				break
			}
			nsec, rangeErrString, err = parseNanoseconds(value, ndigit)
			value = value[ndigit:]
		case stdFracSecond9:
			if len(value) < 2 || !commaOrPeriod(value[0]) ||
				value[1] < '0' || '9' < value[1] {
				// The value holds no fraction of a second.
				break
			}
			i := 0
			for i+1 < len(value) && '0' <= value[i+1] && value[i+1] <= '9' {
				i++
			}
			nsec, rangeErrString, err = parseNanoseconds(value, 1+i)
			value = value[1+i:]
		}
		if rangeErrString != "" {
			return Time{}, newParseError(alayout, avalue, stdstr, value,
				": "+rangeErrString+" out of range")
		}
		if err != nil {
			return Time{}, newParseError(alayout, avalue, stdstr, hold, "")
		}
	}
	if pmSet && hour < 12 {
		hour += 12
	} else if amSet && hour == 12 {
		hour = 0
	}

	if yday >= 0 {
		d := 0
		m := 0
		if isLeap(year) {
			if yday == 31+29 {
				m = int(February)
				d = 29
			} else if yday > 31+29 {
				yday--
			}
		}
		if yday < 1 || yday > 365 {
			return Time{}, newParseError(alayout, avalue, "", value,
				": day-of-year out of range")
		}
		if m == 0 {
			m = (yday-1)/31 + 1
			if daysBefore(Month(m+1)) < yday {
				m++
			}
			d = yday - daysBefore(Month(m))
		}
		// The month and the day of the layout must agree with the day of the
		// year.
		if month >= 0 && month != m {
			return Time{}, newParseError(alayout, avalue, "", value,
				": day-of-year does not match month")
		}
		month = m
		if day >= 0 && day != d {
			return Time{}, newParseError(alayout, avalue, "", value,
				": day-of-year does not match day")
		}
		day = d
	} else {
		if month < 0 {
			month = int(January)
		}
		if day < 0 {
			day = 1
		}
	}

	if day < 1 || day > daysIn(Month(month), year) {
		return Time{}, newParseError(alayout, avalue, "", value, ": day out of range")
	}

	if z != nil {
		return Date(year, Month(month), day, hour, min, sec, nsec, z), nil
	}

	if zoneOffset != -1 {
		t := Date(year, Month(month), day, hour, min, sec, nsec, UTC)
		t.sec -= int64(zoneOffset)
		// The local zone takes the time when it holds that offset at the
		// instant.
		offset, name := local.lookup(t.unixSec())
		if offset == zoneOffset && (zoneName == "" || name == zoneName) {
			t.loc = local
			return t, nil
		}
		t.loc = FixedZone(zoneName, zoneOffset)
		return t, nil
	}

	if zoneName != "" {
		t := Date(year, Month(month), day, hour, min, sec, nsec, UTC)
		if offset, ok := local.lookupName(zoneName, t.unixSec()); ok {
			t.sec -= int64(offset)
			t.loc = local
			return t, nil
		}
		// A name that the local zone does not hold gives a zone of a fixed
		// offset. GMT names its own offset, and each other name gives 0.
		offset := 0
		if len(zoneName) > 3 && zoneName[:3] == "GMT" {
			offset, _ = atoi(zoneName[3:])
			offset *= 3600
		}
		t.loc = FixedZone(zoneName, offset)
		return t, nil
	}

	return Date(year, Month(month), day, hour, min, sec, nsec, defaultLocation), nil
}

func newParseError(layout, value, layoutElem, valueElem, message string) *ParseError {
	return &ParseError{layout, value, layoutElem, valueElem, message}
}
