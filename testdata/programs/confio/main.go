// The program proves the package io against real Go: each copy function, each
// read helper, the composed readers and writers, SectionReader, Pipe with a
// goroutine, the sentinel errors in each position, and the interfaces of io/fs
// over a fstest.MapFS.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
	"testing/fstest"
	"time"
)

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

// shortReader gives at most n bytes for each Read, and it gives io.EOF with
// the last bytes when withEOF is set.
type shortReader struct {
	text    string
	n       int
	withEOF bool
}

func (r *shortReader) Read(p []byte) (int, error) {
	if len(r.text) == 0 {
		return 0, io.EOF
	}
	k := min(r.n, len(r.text), len(p))
	copy(p, r.text[:k])
	r.text = r.text[k:]
	if r.withEOF && len(r.text) == 0 {
		return k, io.EOF
	}
	return k, nil
}

// errReader fails after the text with an error of its own.
type errReader struct {
	text string
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if len(r.text) == 0 {
		return 0, r.err
	}
	n := copy(p, r.text)
	r.text = r.text[n:]
	return n, nil
}

// shortWriter takes at most limit bytes and then fails with no error text,
// thus io.Copy must report io.ErrShortWrite.
type shortWriter struct {
	limit int
	buf   bytes.Buffer
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if w.buf.Len()+len(p) > w.limit {
		n := w.limit - w.buf.Len()
		w.buf.Write(p[:n])
		return n, nil
	}
	return w.buf.Write(p)
}

// countWriter counts the calls of Write.
type countWriter struct {
	calls int
	bytes int
}

func (w *countWriter) Write(p []byte) (int, error) {
	w.calls++
	w.bytes += len(p)
	return len(p), nil
}

var errCustom = errors.New("a custom failure")

func copies() {
	var dst bytes.Buffer
	n, err := io.Copy(&dst, strings.NewReader("copy me"))
	row("Copy", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.Copy(&dst, &shortReader{text: "in small steps", n: 3})
	row("Copy of a short reader", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.Copy(&dst, &shortReader{text: "eof with data", n: 4, withEOF: true})
	row("Copy with EOF and data", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.Copy(&dst, &errReader{text: "partial", err: errCustom})
	row("Copy with an error", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	sw := &shortWriter{limit: 5}
	n, err = io.Copy(sw, strings.NewReader("too long"))
	row("Copy to a short writer", fmt.Sprintf("%d %v %q", n, err, sw.buf.String()))
	row("is ErrShortWrite", errors.Is(err, io.ErrShortWrite))
	dst.Reset()
	n, err = io.CopyN(&dst, strings.NewReader("0123456789"), 4)
	row("CopyN", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.CopyN(&dst, strings.NewReader("012"), 4)
	row("CopyN short", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.CopyN(&dst, strings.NewReader("012"), 0)
	row("CopyN 0", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.CopyN(&dst, strings.NewReader("012"), -1)
	row("CopyN -1", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.CopyBuffer(&dst, &shortReader{text: "buffered copy", n: 5}, make([]byte, 2))
	row("CopyBuffer", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	dst.Reset()
	n, err = io.CopyBuffer(&dst, strings.NewReader("nil buf"), nil)
	row("CopyBuffer nil", fmt.Sprintf("%d %v %q", n, err, dst.String()))
	catch("CopyBuffer empty", func() { io.CopyBuffer(&dst, &shortReader{text: "x", n: 1}, []byte{}) })
	cw := &countWriter{}
	n, err = io.Copy(cw, &shortReader{text: strings.Repeat("a", 10), n: 3})
	row("Copy calls", fmt.Sprintf("%d %v calls=%d bytes=%d", n, err, cw.calls, cw.bytes))
	dst.Reset()
	src := bytes.NewBufferString("WriterTo path")
	n, err = io.Copy(&dst, src)
	row("Copy via WriterTo", fmt.Sprintf("%d %v %q rest=%d", n, err, dst.String(), src.Len()))
}

func reads() {
	data, err := io.ReadAll(strings.NewReader("all of it"))
	row("ReadAll", fmt.Sprintf("%q %v", data, err))
	data, err = io.ReadAll(&shortReader{text: strings.Repeat("ab", 400), n: 7})
	row("ReadAll of a short reader", fmt.Sprintf("%d %v", len(data), err))
	data, err = io.ReadAll(&errReader{text: "part", err: errCustom})
	row("ReadAll with an error", fmt.Sprintf("%q %v", data, err))
	data, err = io.ReadAll(strings.NewReader(""))
	row("ReadAll of empty", fmt.Sprintf("%q %v nil=%v", data, err, data == nil))
	buf := make([]byte, 5)
	n, err := io.ReadFull(strings.NewReader("0123456789"), buf)
	row("ReadFull", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = io.ReadFull(strings.NewReader("012"), buf)
	row("ReadFull short", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	row("is ErrUnexpectedEOF", errors.Is(err, io.ErrUnexpectedEOF))
	n, err = io.ReadFull(strings.NewReader(""), buf)
	row("ReadFull of empty", fmt.Sprintf("%d %v", n, err))
	row("is EOF", errors.Is(err, io.EOF))
	n, err = io.ReadFull(&shortReader{text: "abcdefg", n: 2}, buf)
	row("ReadFull of a short reader", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = io.ReadFull(&errReader{text: "ab", err: errCustom}, buf)
	row("ReadFull with an error", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = io.ReadFull(strings.NewReader("abc"), nil)
	row("ReadFull into nil", fmt.Sprintf("%d %v", n, err))
	n, err = io.ReadAtLeast(strings.NewReader("0123456789"), buf, 3)
	row("ReadAtLeast", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = io.ReadAtLeast(&shortReader{text: "0123456789", n: 2}, buf, 3)
	row("ReadAtLeast short reader", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = io.ReadAtLeast(strings.NewReader("01"), buf, 3)
	row("ReadAtLeast short", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = io.ReadAtLeast(strings.NewReader(""), buf, 3)
	row("ReadAtLeast of empty", fmt.Sprintf("%d %v", n, err))
	n, err = io.ReadAtLeast(strings.NewReader("abc"), buf, 10)
	row("ReadAtLeast over the buffer", fmt.Sprintf("%d %v", n, err))
	row("is ErrShortBuffer", errors.Is(err, io.ErrShortBuffer))
	n, err = io.ReadAtLeast(strings.NewReader("abc"), buf, 0)
	row("ReadAtLeast 0", fmt.Sprintf("%d %v", n, err))
	m, err := io.WriteString(&bytes.Buffer{}, "text")
	row("WriteString", fmt.Sprint(m, " ", err))
	cw := &countWriter{}
	m, err = io.WriteString(cw, "no StringWriter")
	row("WriteString plain", fmt.Sprintf("%d %v calls=%d", m, err, cw.calls))
	m, err = io.Discard.Write([]byte("gone"))
	row("Discard Write", fmt.Sprint(m, " ", err))
	m, err = io.WriteString(io.Discard, "gone too")
	row("Discard WriteString", fmt.Sprint(m, " ", err))
	k, err := io.Copy(io.Discard, &shortReader{text: strings.Repeat("x", 1000), n: 100})
	row("Copy to Discard", fmt.Sprint(k, " ", err))
	rc := io.NopCloser(strings.NewReader("closer"))
	data, _ = io.ReadAll(rc)
	row("NopCloser", fmt.Sprintf("%q %v", data, rc.Close()))
	_, isWriterTo := rc.(io.WriterTo)
	row("NopCloser keeps WriterTo", isWriterTo)
	_, isWriterTo = io.NopCloser(&shortReader{}).(io.WriterTo)
	row("NopCloser plain", isWriterTo)
	n, err = io.ReadFull(io.MultiReader(), buf)
	row("empty MultiReader", fmt.Sprint(n, " ", err))
}

func composed() {
	lr := io.LimitReader(strings.NewReader("0123456789"), 4)
	data, err := io.ReadAll(lr)
	row("LimitReader", fmt.Sprintf("%q %v", data, err))
	lr = io.LimitReader(strings.NewReader("012"), 10)
	data, err = io.ReadAll(lr)
	row("LimitReader long", fmt.Sprintf("%q %v", data, err))
	lr = io.LimitReader(strings.NewReader("012"), 0)
	data, err = io.ReadAll(lr)
	row("LimitReader 0", fmt.Sprintf("%q %v", data, err))
	lr = io.LimitReader(strings.NewReader("012"), -1)
	n, err := lr.Read(make([]byte, 2))
	row("LimitReader -1", fmt.Sprint(n, " ", err))
	limited := &io.LimitedReader{R: strings.NewReader("abcdef"), N: 3}
	limited.Read(make([]byte, 2))
	row("LimitedReader N", limited.N)
	mr := io.MultiReader(strings.NewReader("one "), strings.NewReader(""), strings.NewReader("two "), &shortReader{text: "three", n: 2})
	data, err = io.ReadAll(mr)
	row("MultiReader", fmt.Sprintf("%q %v", data, err))
	mr = io.MultiReader(strings.NewReader("ab"), &errReader{text: "cd", err: errCustom}, strings.NewReader("ef"))
	data, err = io.ReadAll(mr)
	row("MultiReader with an error", fmt.Sprintf("%q %v", data, err))
	mr = io.MultiReader(strings.NewReader("abc"), strings.NewReader("def"))
	buf := make([]byte, 4)
	n, err = mr.Read(buf)
	row("MultiReader Read 1", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = mr.Read(buf)
	row("MultiReader Read 2", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = mr.Read(buf)
	row("MultiReader Read 3", fmt.Sprintf("%d %v", n, err))
	inner := io.MultiReader(strings.NewReader("x"), strings.NewReader("y"))
	outer := io.MultiReader(inner, strings.NewReader("z"))
	data, _ = io.ReadAll(outer)
	row("MultiReader nested", fmt.Sprintf("%q", data))
	var a, b bytes.Buffer
	cw := &countWriter{}
	mw := io.MultiWriter(&a, &b, cw)
	n, err = mw.Write([]byte("both"))
	row("MultiWriter", fmt.Sprintf("%d %v %q %q calls=%d", n, err, a.String(), b.String(), cw.calls))
	n, err = io.WriteString(mw, " and a string")
	row("MultiWriter WriteString", fmt.Sprintf("%d %v %q", n, err, a.String()))
	sw := &shortWriter{limit: 2}
	mw = io.MultiWriter(&a, sw, &b)
	n, err = mw.Write([]byte("stop"))
	row("MultiWriter short", fmt.Sprintf("%d %v a=%q b=%q", n, err, a.String(), b.String()))
	var seen bytes.Buffer
	tr := io.TeeReader(strings.NewReader("tee this"), &seen)
	data, err = io.ReadAll(tr)
	row("TeeReader", fmt.Sprintf("%q %v seen=%q", data, err, seen.String()))
	seen.Reset()
	tr = io.TeeReader(&errReader{text: "part", err: errCustom}, &seen)
	data, err = io.ReadAll(tr)
	row("TeeReader with an error", fmt.Sprintf("%q %v seen=%q", data, err, seen.String()))
	tr = io.TeeReader(strings.NewReader("short"), &shortWriter{limit: 2})
	n, err = tr.Read(buf)
	row("TeeReader short writer", fmt.Sprintf("%d %v", n, err))
	sr := io.NewSectionReader(strings.NewReader("0123456789"), 2, 5)
	data, err = io.ReadAll(sr)
	row("SectionReader", fmt.Sprintf("%q %v", data, err))
	row("SectionReader Size", sr.Size())
	pos, err := sr.Seek(1, io.SeekStart)
	row("SectionReader Seek", fmt.Sprint(pos, " ", err))
	n, err = sr.Read(buf)
	row("SectionReader Read", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = sr.Read(buf)
	row("SectionReader Read at end", fmt.Sprint(n, " ", err))
	pos, err = sr.Seek(-2, io.SeekEnd)
	row("SectionReader Seek end", fmt.Sprint(pos, " ", err))
	n, err = sr.ReadAt(buf, 3)
	row("SectionReader ReadAt", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = sr.ReadAt(buf[:2], 1)
	row("SectionReader ReadAt full", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = sr.ReadAt(buf, 5)
	row("SectionReader ReadAt at end", fmt.Sprint(n, " ", err))
	n, err = sr.ReadAt(buf, -1)
	row("SectionReader ReadAt -1", fmt.Sprint(n, " ", err))
	_, err = sr.Seek(-1, io.SeekStart)
	row("SectionReader Seek -1", err)
	_, err = sr.Seek(0, 5)
	row("SectionReader Seek bad", err)
	base, off, k := sr.Outer()
	row("SectionReader Outer", fmt.Sprintf("%v %d %d", base != nil, off, k))
	sr = io.NewSectionReader(strings.NewReader("short"), 3, 10)
	data, err = io.ReadAll(sr)
	row("SectionReader past the end", fmt.Sprintf("%q %v", data, err))
	ow := io.NewOffsetWriter(&sliceAt{buf: make([]byte, 10)}, 2)
	n, err = ow.Write([]byte("abc"))
	row("OffsetWriter Write", fmt.Sprint(n, " ", err))
	n, err = ow.WriteAt([]byte("Z"), 6)
	row("OffsetWriter WriteAt", fmt.Sprint(n, " ", err))
	pos, err = ow.Seek(1, io.SeekCurrent)
	row("OffsetWriter Seek", fmt.Sprint(pos, " ", err))
	_, err = ow.WriteAt([]byte("x"), -1)
	row("OffsetWriter WriteAt -1", err)
	var ra io.ReaderAt = strings.NewReader("reader at")
	n, err = ra.ReadAt(buf, 7)
	row("ReaderAt", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	var rs io.ReadSeeker = strings.NewReader("seek")
	rs.Seek(2, io.SeekStart)
	data, _ = io.ReadAll(rs)
	row("ReadSeeker", fmt.Sprintf("%q", data))
	br := io.ByteReader(strings.NewReader("b"))
	c, err := br.ReadByte()
	row("ByteReader", fmt.Sprintf("%q %v", c, err))
	rr := io.RuneReader(strings.NewReader("日"))
	r, size, err := rr.ReadRune()
	row("RuneReader", fmt.Sprintf("%q %d %v", r, size, err))
	var sc io.RuneScanner = strings.NewReader("ab")
	sc.ReadRune()
	sc.UnreadRune()
	r, _, _ = sc.ReadRune()
	row("RuneScanner", fmt.Sprintf("%q", r))
}

// sliceAt holds a fixed slice with WriteAt, thus OffsetWriter has a target.
type sliceAt struct{ buf []byte }

func (s *sliceAt) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 || int(off) >= len(s.buf) {
		return 0, errCustom
	}
	return copy(s.buf[off:], p), nil
}

func pipes() {
	pr, pw := io.Pipe()
	done := make(chan string)
	go func() {
		data, err := io.ReadAll(pr)
		done <- fmt.Sprintf("%q %v", data, err)
	}()
	for i := 0; i < 3; i++ {
		fmt.Fprintf(pw, "chunk %d;", i)
	}
	pw.Close()
	row("Pipe read all", <-done)
	n, err := pw.Write([]byte("after close"))
	row("Pipe write after close", fmt.Sprint(n, " ", err))
	row("is ErrClosedPipe", errors.Is(err, io.ErrClosedPipe))

	pr, pw = io.Pipe()
	go func() {
		pw.Write([]byte("first"))
		pw.CloseWithError(errCustom)
	}()
	data, err := io.ReadAll(pr)
	row("Pipe CloseWithError", fmt.Sprintf("%q %v", data, err))

	pr, pw = io.Pipe()
	go func() {
		pr.CloseWithError(errors.New("the reader left"))
	}()
	n, err = pw.Write([]byte("nobody reads"))
	row("Pipe reader closed", fmt.Sprint(n, " ", err))

	pr, pw = io.Pipe()
	go func() {
		pw.Write([]byte("ab"))
		pw.Write([]byte("cd"))
		pw.Close()
	}()
	buf := make([]byte, 1)
	var parts []string
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			parts = append(parts, string(buf[:n]))
		}
		if err != nil {
			parts = append(parts, err.Error())
			break
		}
	}
	row("Pipe byte by byte", parts)
	pr, pw = io.Pipe()
	go func() {
		io.WriteString(pw, "via WriteString")
		pw.Close()
	}()
	data, _ = io.ReadAll(pr)
	row("Pipe WriteString", fmt.Sprintf("%q", data))
	row("Pipe Close twice", fmt.Sprint(pw.Close(), " ", pr.Close()))
}

func sentinels() {
	row("EOF", io.EOF)
	row("ErrUnexpectedEOF", io.ErrUnexpectedEOF)
	row("ErrShortWrite", io.ErrShortWrite)
	row("ErrShortBuffer", io.ErrShortBuffer)
	row("ErrNoProgress", io.ErrNoProgress)
	row("ErrClosedPipe", io.ErrClosedPipe)
	row("EOF is not ErrUnexpectedEOF", errors.Is(io.EOF, io.ErrUnexpectedEOF))
	wrapped := fmt.Errorf("read failed: %w", io.EOF)
	row("wrapped EOF", fmt.Sprintf("%v is=%v", wrapped, errors.Is(wrapped, io.EOF)))
	row("SeekStart", io.SeekStart)
	row("SeekCurrent", io.SeekCurrent)
	row("SeekEnd", io.SeekEnd)
}

func fsys() {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	m := fstest.MapFS{
		"hello.txt":       {Data: []byte("hello, world\n"), Mode: 0o644, ModTime: now},
		"dir/a.txt":       {Data: []byte("aaa"), Mode: 0o600, ModTime: now},
		"dir/b.txt":       {Data: []byte("bb"), Mode: 0o644, ModTime: now},
		"dir/sub/c.md":    {Data: []byte("# c"), Mode: 0o644, ModTime: now},
		"empty":           {Mode: fs.ModeDir | 0o755, ModTime: now},
		"link":            {Data: []byte("hello.txt"), Mode: fs.ModeSymlink | 0o777, ModTime: now},
		"exec.sh":         {Data: []byte("#!/bin/sh\n"), Mode: 0o755, ModTime: now},
		"z/deep/x/y/z.go": {Data: []byte("package z"), Mode: 0o644, ModTime: now},
	}
	data, err := fs.ReadFile(m, "hello.txt")
	row("ReadFile", fmt.Sprintf("%q %v", data, err))
	_, err = fs.ReadFile(m, "missing.txt")
	row("ReadFile missing", err)
	row("is ErrNotExist", errors.Is(err, fs.ErrNotExist))
	var pe *fs.PathError
	row("as PathError", fmt.Sprintf("%v op=%q path=%q", errors.As(err, &pe), pe.Op, pe.Path))
	_, err = fs.ReadFile(m, "../bad")
	row("ReadFile invalid path", err)
	row("is ErrInvalid", errors.Is(err, fs.ErrInvalid))
	f, err := m.Open("dir/a.txt")
	row("Open", err)
	info, err := f.Stat()
	row("Stat of a file", fmt.Sprintf("%v %s %d %v %v %v", err, info.Name(), info.Size(), info.Mode(), info.IsDir(), info.ModTime().Equal(now)))
	buf := make([]byte, 2)
	n, err := f.Read(buf)
	row("Read", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	row("Close", f.Close())
	entries, err := fs.ReadDir(m, "dir")
	row("ReadDir err", err)
	for _, e := range entries {
		info, _ := e.Info()
		fmt.Printf("ReadDir entry %-8s dir=%-5v type=%-10v mode=%v size=%d\n", e.Name(), e.IsDir(), e.Type(), info.Mode(), info.Size())
	}
	entries, err = fs.ReadDir(m, ".")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	row("ReadDir root", fmt.Sprintf("%v %v", names, err))
	_, err = fs.ReadDir(m, "hello.txt")
	row("ReadDir of a file", err)
	_, err = fs.ReadDir(m, "nope")
	row("ReadDir missing", err)
	info, err = fs.Stat(m, "dir")
	row("Stat of a dir", fmt.Sprintf("%v %s %v %v", err, info.Name(), info.Mode(), info.IsDir()))
	info, err = fs.Stat(m, "empty")
	row("Stat of an empty dir", fmt.Sprintf("%v %s %v", err, info.Name(), info.Mode()))
	info, err = fs.Stat(m, "link")
	row("Stat of a link", fmt.Sprintf("%v %s %v", err, info.Name(), info.Mode()))
	info, err = fs.Stat(m, ".")
	row("Stat of root", fmt.Sprintf("%v %q %v", err, info.Name(), info.IsDir()))
	info, _ = fs.Stat(m, "exec.sh")
	row("mode text", info.Mode().String())
	row("mode Perm", info.Mode().Perm())
	row("mode Type", info.Mode().Type())
	row("mode IsRegular", info.Mode().IsRegular())
	row("FormatFileInfo", fs.FormatFileInfo(info))
	entries, _ = fs.ReadDir(m, ".")
	row("FormatDirEntry file", fs.FormatDirEntry(entries[2]))
	row("FormatDirEntry dir", fs.FormatDirEntry(entries[0]))
	var walked []string
	err = fs.WalkDir(m, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		kind := "f"
		if d.IsDir() {
			kind = "d"
		}
		walked = append(walked, kind+":"+path)
		return nil
	})
	row("WalkDir err", err)
	for _, w := range walked {
		fmt.Println("WalkDir", w)
	}
	walked = nil
	err = fs.WalkDir(m, "dir", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && d.Name() == "sub" {
			return fs.SkipDir
		}
		walked = append(walked, path)
		return nil
	})
	row("WalkDir SkipDir", fmt.Sprintf("%v %v", walked, err))
	walked = nil
	err = fs.WalkDir(m, ".", func(path string, d fs.DirEntry, err error) error {
		walked = append(walked, path)
		if path == "dir/a.txt" {
			return fs.SkipAll
		}
		return nil
	})
	row("WalkDir SkipAll", fmt.Sprintf("%v %v", walked, err))
	err = fs.WalkDir(m, "missing", func(path string, d fs.DirEntry, err error) error {
		walked = []string{path, fmt.Sprint(d == nil), err.Error()}
		return err
	})
	row("WalkDir missing", fmt.Sprintf("%v %v", walked, err))
	err = fs.WalkDir(m, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "dir/b.txt" {
			return errCustom
		}
		return nil
	})
	row("WalkDir custom error", err)
	matches, err := fs.Glob(m, "dir/*.txt")
	row("Glob", fmt.Sprintf("%v %v", matches, err))
	matches, err = fs.Glob(m, "*")
	row("Glob root", fmt.Sprintf("%v %v", matches, err))
	matches, err = fs.Glob(m, "*/*/*")
	row("Glob deep", fmt.Sprintf("%v %v", matches, err))
	matches, err = fs.Glob(m, "[")
	row("Glob bad pattern", fmt.Sprintf("%v %v", matches, err))
	matches, err = fs.Glob(m, "none*")
	row("Glob no match", fmt.Sprintf("%v %v nil=%v", matches, err, matches == nil))
	sub, err := fs.Sub(m, "dir")
	row("Sub err", err)
	data, err = fs.ReadFile(sub, "sub/c.md")
	row("Sub ReadFile", fmt.Sprintf("%q %v", data, err))
	_, err = fs.ReadFile(sub, "../hello.txt")
	row("Sub escape", err)
	same, _ := fs.Sub(m, ".")
	row("Sub of dot is the same", same != nil)
	_, err = fs.Sub(m, "/abs")
	row("Sub bad dir", err)
	for _, p := range []string{".", "a", "a/b", "/a", "a/", "a//b", "a/./b", "a/../b", "..", "", "a\\b", "日本/語"} {
		fmt.Printf("ValidPath %-8q %v\n", p, fs.ValidPath(p))
	}
	target, err := fs.ReadLink(m, "link")
	row("ReadLink", fmt.Sprintf("%q %v", target, err))
	_, err = fs.ReadLink(m, "hello.txt")
	row("ReadLink of a file", err)
	info, err = fs.Lstat(m, "link")
	row("Lstat of a link", fmt.Sprintf("%v %v", err, info.Mode()))
	row("TestFS", fstest.TestFS(m, "hello.txt", "dir/a.txt", "dir/sub/c.md", "z/deep/x/y/z.go"))
	row("ErrInvalid", fs.ErrInvalid)
	row("ErrPermission", fs.ErrPermission)
	row("ErrExist", fs.ErrExist)
	row("ErrNotExist", fs.ErrNotExist)
	row("ErrClosed", fs.ErrClosed)
	row("SkipDir", fs.SkipDir)
	row("SkipAll", fs.SkipAll)
	modes := []fs.FileMode{0, 0o644, 0o755, fs.ModeDir | 0o755, fs.ModeSymlink | 0o777, fs.ModeNamedPipe, fs.ModeSocket, fs.ModeDevice | fs.ModeCharDevice, fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky | 0o777, fs.ModeAppend | fs.ModeExclusive | fs.ModeTemporary, fs.ModeIrregular, fs.ModeType, fs.ModePerm}
	for _, mode := range modes {
		fmt.Printf("FileMode %#o %-16s type=%-12s perm=%v regular=%v dir=%v\n", uint32(mode), mode, mode.Type(), mode.Perm(), mode.IsRegular(), mode.IsDir())
	}
	var files []string
	for name := range m {
		files = append(files, name)
	}
	sort.Strings(files)
	row("MapFS keys", files)
	entries, _ = fs.ReadDir(m, "z/deep/x/y")
	row("deep dir", entries[0].Name())
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-30s %v\n", label, recover())
	}()
	f()
}

func main() {
	fmt.Println("== the copy functions ==")
	copies()
	fmt.Println("== the read helpers ==")
	reads()
	fmt.Println("== the composed readers and writers ==")
	composed()
	fmt.Println("== Pipe ==")
	pipes()
	fmt.Println("== the sentinels ==")
	sentinels()
	fmt.Println("== io/fs ==")
	fsys()
}
