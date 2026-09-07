// The program proves the package bufio against real Go: Reader with each read
// method over a small buffer, Scanner with each split function and with a
// custom split function, a long line against the token limit, Writer with
// Flush, Available, and Buffered, and ReadWriter.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

// slowReader gives at most 3 bytes for each Read, thus a Reader of bufio must
// fill its buffer in more than 1 call.
type slowReader struct {
	text string
	pos  int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.text) {
		return 0, io.EOF
	}
	n := copy(p, r.text[r.pos:min(r.pos+3, len(r.text))])
	r.pos += n
	return n, nil
}

func reader() {
	text := "first line\nsecond line\r\nthird 日本語 line\nno newline at end"
	r := bufio.NewReaderSize(strings.NewReader(text), 16)
	row("Size", r.Size())
	row("Buffered at start", r.Buffered())
	line, err := r.ReadString('\n')
	row("ReadString", fmt.Sprintf("%q %v", line, err))
	row("Buffered after", r.Buffered())
	bs, err := r.ReadBytes('\n')
	row("ReadBytes", fmt.Sprintf("%q %v", bs, err))
	peek, err := r.Peek(5)
	row("Peek 5", fmt.Sprintf("%q %v", peek, err))
	peek, err = r.Peek(20)
	row("Peek 20 over the buffer", fmt.Sprintf("%q %v", peek, err))
	row("Peek 20 is ErrBufferFull", errors.Is(err, bufio.ErrBufferFull))
	peek, err = r.Peek(-1)
	row("Peek -1", fmt.Sprintf("%q %v", peek, err))
	n, err := r.Discard(6)
	row("Discard 6", fmt.Sprint(n, " ", err))
	ch, size, err := r.ReadRune()
	row("ReadRune", fmt.Sprintf("%q %d %v", ch, size, err))
	err = r.UnreadRune()
	row("UnreadRune", err)
	err = r.UnreadRune()
	row("UnreadRune twice", err)
	ch, size, _ = r.ReadRune()
	row("ReadRune again", fmt.Sprintf("%q %d", ch, size))
	c, err := r.ReadByte()
	row("ReadByte", fmt.Sprintf("%q %v", c, err))
	err = r.UnreadByte()
	row("UnreadByte", err)
	c, _ = r.ReadByte()
	row("ReadByte again", fmt.Sprintf("%q", c))
	err = r.UnreadRune()
	row("UnreadRune after ReadByte", err)
	lineBytes, isPrefix, err := r.ReadLine()
	row("ReadLine", fmt.Sprintf("%q %v %v", lineBytes, isPrefix, err))
	lineBytes, isPrefix, err = r.ReadLine()
	row("ReadLine long", fmt.Sprintf("%q %v %v", lineBytes, isPrefix, err))
	lineBytes, isPrefix, err = r.ReadLine()
	row("ReadLine rest", fmt.Sprintf("%q %v %v", lineBytes, isPrefix, err))
	lineBytes, isPrefix, err = r.ReadLine()
	row("ReadLine at end", fmt.Sprintf("%q %v %v", lineBytes, isPrefix, err))
	line, err = r.ReadString('\n')
	row("ReadString at end", fmt.Sprintf("%q %v", line, err))
	n, err = r.Discard(1)
	row("Discard at end", fmt.Sprint(n, " ", err))
	n, err = r.Discard(-1)
	row("Discard -1", fmt.Sprint(n, " ", err))
	_, err = r.ReadByte()
	row("ReadByte at end", err)
	err = r.UnreadByte()
	row("UnreadByte at end", err)

	r = bufio.NewReaderSize(&slowReader{text: "abcdefghij"}, 16)
	row("small size becomes 16", r.Size())
	peek, err = r.Peek(7)
	row("Peek over a slow reader", fmt.Sprintf("%q %v", peek, err))
	buf := make([]byte, 4)
	n, err = r.Read(buf)
	row("Read", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = r.Read(buf)
	row("Read 2", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = r.Read(buf)
	row("Read 3", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = r.Read(buf)
	row("Read 4", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = r.Read(buf)
	row("Read at end", fmt.Sprintf("%d %v", n, err))
	n, err = r.Read(nil)
	row("Read into nil", fmt.Sprintf("%d %v", n, err))

	r = bufio.NewReader(strings.NewReader("a\xffb"))
	ch, size, _ = r.ReadRune()
	row("ReadRune valid", fmt.Sprintf("%q %d", ch, size))
	ch, size, _ = r.ReadRune()
	row("ReadRune invalid", fmt.Sprintf("%q %d", ch, size))
	r.Reset(strings.NewReader("reset text"))
	line, _ = r.ReadString(' ')
	row("Reset", fmt.Sprintf("%q", line))
	var sb strings.Builder
	m, err := r.WriteTo(&sb)
	row("WriteTo", fmt.Sprintf("%d %v %q", m, err, sb.String()))
	row("Buffered after WriteTo", r.Buffered())
	all, err := io.ReadAll(bufio.NewReaderSize(&slowReader{text: strings.Repeat("xy", 50)}, 16))
	row("ReadAll of a slow reader", fmt.Sprint(len(all), " ", err))
	wrapped := bufio.NewReader(strings.NewReader("x"))
	same := bufio.NewReader(wrapped)
	row("NewReader of a Reader", same == wrapped)
	bigger := bufio.NewReaderSize(wrapped, 8192)
	row("NewReaderSize bigger", bigger != wrapped)
	row("bigger Size", bigger.Size())
	row("default Size", bufio.NewReader(strings.NewReader("")).Size())
	row("ErrNegativeCount", bufio.ErrNegativeCount)
	row("ErrInvalidUnreadByte", bufio.ErrInvalidUnreadByte)
	row("ErrInvalidUnreadRune", bufio.ErrInvalidUnreadRune)
	row("ErrBufferFull", bufio.ErrBufferFull)
}

func scanner() {
	text := "alpha beta\ngamma  delta\n\nepsilon\n"
	sc := bufio.NewScanner(strings.NewReader(text))
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("ScanLines", fmt.Sprintf("%q", lines))
	row("ScanLines err", sc.Err())
	row("Text after end", fmt.Sprintf("%q", sc.Text()))
	row("Bytes after end", sc.Bytes() == nil)

	sc = bufio.NewScanner(strings.NewReader("a\r\nb\rc\n\r\nlast"))
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("ScanLines crlf", fmt.Sprintf("%q", lines))

	sc = bufio.NewScanner(strings.NewReader(text))
	sc.Split(bufio.ScanWords)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("ScanWords", fmt.Sprintf("%q", lines))
	sc = bufio.NewScanner(strings.NewReader("  a\u00a0b\u3000c\t\vd  "))
	sc.Split(bufio.ScanWords)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("ScanWords unicode space", fmt.Sprintf("%q", lines))

	sc = bufio.NewScanner(strings.NewReader("a日\xffb"))
	sc.Split(bufio.ScanRunes)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("ScanRunes", fmt.Sprintf("%q", lines))

	sc = bufio.NewScanner(strings.NewReader("a日"))
	sc.Split(bufio.ScanBytes)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("ScanBytes", fmt.Sprintf("%q", lines))

	// A custom split function cuts at each comma, and it drops the space
	// around a field.
	commas := func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, ','); i >= 0 {
			return i + 1, bytes.TrimSpace(data[:i]), nil
		}
		if atEOF {
			return len(data), bytes.TrimSpace(data), nil
		}
		return 0, nil, nil
	}
	sc = bufio.NewScanner(strings.NewReader("one, two ,three,, five"))
	sc.Split(commas)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("custom split", fmt.Sprintf("%q", lines))
	row("custom split err", sc.Err())

	// A split function that stops with an error of its own.
	stopAt := func(data []byte, atEOF bool) (int, []byte, error) {
		adv, tok, err := bufio.ScanLines(data, atEOF)
		if tok != nil && string(tok) == "stop" {
			return 0, nil, errors.New("the stop word")
		}
		return adv, tok, err
	}
	sc = bufio.NewScanner(strings.NewReader("go\nstop\nnever"))
	sc.Split(stopAt)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("split with an error", fmt.Sprintf("%q %v", lines, sc.Err()))

	// A split function that gives ErrFinalToken ends the scan with no error.
	finalToken := func(data []byte, atEOF bool) (int, []byte, error) {
		adv, tok, err := bufio.ScanWords(data, atEOF)
		if tok != nil && string(tok) == "end" {
			return adv, tok, bufio.ErrFinalToken
		}
		return adv, tok, err
	}
	sc = bufio.NewScanner(strings.NewReader("a b end c d"))
	sc.Split(finalToken)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("ErrFinalToken", fmt.Sprintf("%q %v", lines, sc.Err()))

	// A long line against the token limit.
	long := strings.Repeat("x", bufio.MaxScanTokenSize+10)
	sc = bufio.NewScanner(strings.NewReader("short\n" + long + "\nafter"))
	lines = nil
	for sc.Scan() {
		lines = append(lines, fmt.Sprint(len(sc.Text())))
	}
	row("long line", fmt.Sprintf("%v %v", lines, sc.Err()))
	row("long line is ErrTooLong", errors.Is(sc.Err(), bufio.ErrTooLong))
	sc = bufio.NewScanner(strings.NewReader("short\n" + long + "\nafter"))
	sc.Buffer(make([]byte, 100), bufio.MaxScanTokenSize*2)
	lines = nil
	for sc.Scan() {
		lines = append(lines, fmt.Sprint(len(sc.Text())))
	}
	row("long line with Buffer", fmt.Sprintf("%v %v", lines, sc.Err()))
	sc = bufio.NewScanner(strings.NewReader("abcdefghij"))
	sc.Buffer(make([]byte, 4), 4)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("tiny Buffer", fmt.Sprintf("%q %v", lines, sc.Err()))
	sc = bufio.NewScanner(strings.NewReader("ab\ncd"))
	sc.Buffer(make([]byte, 4), 4)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("tiny Buffer that fits", fmt.Sprintf("%q %v", lines, sc.Err()))

	sc = bufio.NewScanner(strings.NewReader(""))
	row("Scan of empty", sc.Scan())
	row("Scan of empty err", sc.Err())
	sc = bufio.NewScanner(strings.NewReader("\n"))
	row("Scan of a newline", sc.Scan())
	row("Text of a newline", fmt.Sprintf("%q", sc.Text()))
	row("Scan again", sc.Scan())
	sc = bufio.NewScanner(&slowReader{text: "slow line one\nslow line two"})
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("slow reader", fmt.Sprintf("%q", lines))
	sc = bufio.NewScanner(strings.NewReader("x"))
	sc.Scan()
	catch("Split after Scan", func() { sc.Split(bufio.ScanWords) })
	catch("Buffer after Scan", func() { sc.Buffer(nil, 10) })
	row("MaxScanTokenSize", bufio.MaxScanTokenSize)
	row("ErrTooLong", bufio.ErrTooLong)
	row("ErrNegativeAdvance", bufio.ErrNegativeAdvance)
	row("ErrAdvanceTooFar", bufio.ErrAdvanceTooFar)
	row("ErrBadReadCount", bufio.ErrBadReadCount)
	row("ErrFinalToken", bufio.ErrFinalToken)
	bad := func(data []byte, atEOF bool) (int, []byte, error) {
		return -1, nil, nil
	}
	sc = bufio.NewScanner(strings.NewReader("x"))
	sc.Split(bad)
	row("negative advance", fmt.Sprintf("%v %v", sc.Scan(), sc.Err()))
	far := func(data []byte, atEOF bool) (int, []byte, error) {
		return len(data) + 5, data, nil
	}
	sc = bufio.NewScanner(strings.NewReader("x"))
	sc.Split(far)
	row("advance too far", fmt.Sprintf("%v %v", sc.Scan(), sc.Err()))
	upper := func(data []byte, atEOF bool) (int, []byte, error) {
		adv, tok, err := bufio.ScanWords(data, atEOF)
		if tok != nil {
			tok = bytes.Map(unicode.ToUpper, tok)
		}
		return adv, tok, err
	}
	sc = bufio.NewScanner(strings.NewReader("mixed Case words"))
	sc.Split(upper)
	lines = nil
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	row("split with a map", fmt.Sprintf("%q", lines))
	adv, tok, err := bufio.ScanLines([]byte("abc"), false)
	row("ScanLines direct", fmt.Sprintf("%d %q %v", adv, tok, err))
	adv, tok, err = bufio.ScanLines([]byte("abc"), true)
	row("ScanLines direct at EOF", fmt.Sprintf("%d %q %v", adv, tok, err))
	adv, tok, err = bufio.ScanLines([]byte("abc\r\nd"), false)
	row("ScanLines direct crlf", fmt.Sprintf("%d %q %v", adv, tok, err))
	adv, tok, err = bufio.ScanWords([]byte("  ab  "), false)
	row("ScanWords direct", fmt.Sprintf("%d %q %v", adv, tok, err))
	adv, tok, err = bufio.ScanRunes([]byte("\xe6\x97"), false)
	row("ScanRunes partial", fmt.Sprintf("%d %q %v", adv, tok, err))
	adv, tok, err = bufio.ScanRunes([]byte("\xe6\x97"), true)
	row("ScanRunes partial at EOF", fmt.Sprintf("%d %q %v", adv, tok, err))
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-30s %v\n", label, recover())
	}()
	f()
}

// failWriter fails after the byte limit, thus a Writer of bufio must report
// the error and keep the count.
type failWriter struct {
	limit   int
	written int
}

func (w *failWriter) Write(p []byte) (int, error) {
	if w.written+len(p) > w.limit {
		n := w.limit - w.written
		w.written = w.limit
		return n, errors.New("the disk is full")
	}
	w.written += len(p)
	return len(p), nil
}

func writer() {
	var out bytes.Buffer
	w := bufio.NewWriterSize(&out, 16)
	row("Size", w.Size())
	row("Available", w.Available())
	row("Buffered", w.Buffered())
	n, err := w.WriteString("hello")
	row("WriteString", fmt.Sprint(n, " ", err))
	row("Buffered after", w.Buffered())
	row("Available after", w.Available())
	row("target before Flush", fmt.Sprintf("%q", out.String()))
	err = w.WriteByte('!')
	row("WriteByte", err)
	n, err = w.WriteRune('世')
	row("WriteRune", fmt.Sprint(n, " ", err))
	n, err = w.WriteRune(-1)
	row("WriteRune invalid", fmt.Sprint(n, " ", err))
	row("Buffered", w.Buffered())
	n, err = w.Write([]byte("0123456789"))
	row("Write over the buffer", fmt.Sprint(n, " ", err))
	row("Buffered after overflow", w.Buffered())
	row("target after overflow", fmt.Sprintf("%q", out.String()))
	err = w.Flush()
	row("Flush", err)
	row("Buffered after Flush", w.Buffered())
	row("target after Flush", fmt.Sprintf("%q", out.String()))
	n, err = w.Write([]byte(strings.Repeat("z", 40)))
	row("Write of a large slice", fmt.Sprint(n, " ", err))
	row("Buffered after large", w.Buffered())
	row("target len", out.Len())
	buf := w.AvailableBuffer()
	buf = append(buf, "avail"...)
	w.Write(buf)
	w.Flush()
	row("AvailableBuffer", fmt.Sprintf("%q", out.String()[out.Len()-5:]))
	m, err := w.ReadFrom(strings.NewReader("from a reader"))
	row("ReadFrom", fmt.Sprint(m, " ", err))
	w.Flush()
	row("target tail", fmt.Sprintf("%q", out.String()[out.Len()-13:]))
	out.Reset()
	w.Reset(&out)
	row("Buffered after Reset", w.Buffered())
	fmt.Fprintf(w, "%d-%s", 7, "x")
	w.Flush()
	row("Fprintf", out.String())

	fw := &failWriter{limit: 20}
	w = bufio.NewWriterSize(fw, 8)
	n, err = w.WriteString("12345678")
	row("fail WriteString 1", fmt.Sprint(n, " ", err))
	n, err = w.WriteString("12345678")
	row("fail WriteString 2", fmt.Sprint(n, " ", err))
	n, err = w.WriteString("12345678")
	row("fail WriteString 3", fmt.Sprint(n, " ", err))
	err = w.Flush()
	row("fail Flush", err)
	n, err = w.WriteString("more")
	row("fail after the error", fmt.Sprint(n, " ", err))
	row("fail written", fw.written)
	row("default Size", bufio.NewWriter(&out).Size())
	inner := bufio.NewWriter(&out)
	row("NewWriter of a Writer", bufio.NewWriter(inner) == inner)
	row("NewWriterSize bigger", bufio.NewWriterSize(inner, 8192) != inner)
	row("NewWriterSize 0", bufio.NewWriterSize(&out, 0).Size())
}

func readWriter() {
	var in bytes.Buffer
	in.WriteString("ping\n")
	var out bytes.Buffer
	rw := bufio.NewReadWriter(bufio.NewReader(&in), bufio.NewWriter(&out))
	line, err := rw.ReadString('\n')
	row("ReadWriter ReadString", fmt.Sprintf("%q %v", line, err))
	rw.WriteString("pong\n")
	row("before Flush", fmt.Sprintf("%q", out.String()))
	rw.Flush()
	row("after Flush", fmt.Sprintf("%q", out.String()))
	row("Buffered", rw.Reader.Buffered())
	row("Available", rw.Writer.Available())
	var rwIface io.ReadWriter = rw
	fmt.Fprint(rwIface, "via iface")
	rw.Flush()
	row("as io.ReadWriter", fmt.Sprintf("%q", out.String()))
}

func main() {
	fmt.Println("== Reader ==")
	reader()
	fmt.Println("== Scanner ==")
	scanner()
	fmt.Println("== Writer ==")
	writer()
	fmt.Println("== ReadWriter ==")
	readWriter()
}
