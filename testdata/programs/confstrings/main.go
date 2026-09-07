// The program proves each function of the package strings against real Go. It
// reads the search functions, the split functions, the trim functions, the
// case functions with a rune of a special case, Builder, Reader, Replacer,
// and the 3 iterator functions of Go 1.24.
package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
)

func row(label string, value any) {
	fmt.Printf("%-28s %v\n", label, value)
}

func rowq(label string, value string) {
	fmt.Printf("%-28s %q\n", label, value)
}

func searches() {
	s := "chicken and chips and dips"
	row("Index", strings.Index(s, "and"))
	row("Index absent", strings.Index(s, "beef"))
	row("Index empty", strings.Index(s, ""))
	row("Index of a long sep", strings.Index("ab", "abc"))
	row("LastIndex", strings.LastIndex(s, "and"))
	row("LastIndex empty", strings.LastIndex(s, ""))
	row("LastIndex absent", strings.LastIndex(s, "zz"))
	row("IndexByte", strings.IndexByte(s, 'k'))
	row("IndexByte absent", strings.IndexByte(s, 'z'))
	row("LastIndexByte", strings.LastIndexByte(s, 'd'))
	row("IndexRune", strings.IndexRune("chicken 日本", '本'))
	row("IndexRune absent", strings.IndexRune("chicken", 'x'))
	row("IndexRune invalid", strings.IndexRune("a\xffb", unicode.ReplacementChar))
	row("IndexRune out of range", strings.IndexRune("abc", -1))
	row("IndexAny", strings.IndexAny(s, "pd"))
	row("IndexAny absent", strings.IndexAny(s, "xyz"))
	row("IndexAny empty chars", strings.IndexAny(s, ""))
	row("LastIndexAny", strings.LastIndexAny(s, "ck"))
	row("IndexFunc", strings.IndexFunc("abc123", unicode.IsDigit))
	row("IndexFunc absent", strings.IndexFunc("abc", unicode.IsDigit))
	row("LastIndexFunc", strings.LastIndexFunc("a1b2c", unicode.IsDigit))
	row("Contains", strings.Contains(s, "chips"))
	row("Contains empty", strings.Contains(s, ""))
	row("Contains absent", strings.Contains(s, "fish"))
	row("ContainsAny", strings.ContainsAny(s, "xyzk"))
	row("ContainsAny empty", strings.ContainsAny(s, ""))
	row("ContainsRune", strings.ContainsRune("日本語", '本'))
	row("ContainsFunc", strings.ContainsFunc("abc", unicode.IsUpper))
	row("HasPrefix", strings.HasPrefix(s, "chick"))
	row("HasPrefix empty", strings.HasPrefix(s, ""))
	row("HasSuffix", strings.HasSuffix(s, "dips"))
	row("HasSuffix long", strings.HasSuffix("a", "ab"))
	row("Count", strings.Count(s, "and"))
	row("Count of empty", strings.Count("five", ""))
	row("Count of empty on empty", strings.Count("", ""))
	row("Count overlap", strings.Count("aaaa", "aa"))
	row("Compare less", strings.Compare("a", "b"))
	row("Compare equal", strings.Compare("a", "a"))
	row("Compare more", strings.Compare("b", "a"))
	row("Compare prefix", strings.Compare("ab", "abc"))
	row("Compare empty", strings.Compare("", ""))
}

func splits() {
	row("Split", strings.Split("a,b,c", ","))
	row("Split of empty", len(strings.Split("", ",")))
	row("Split empty sep", strings.Split("日本語", ""))
	row("Split no match", strings.Split("abc", ","))
	row("Split trailing sep", fmt.Sprintf("%q", strings.Split("a,b,", ",")))
	row("Split leading sep", fmt.Sprintf("%q", strings.Split(",a,b", ",")))
	row("Split only seps", fmt.Sprintf("%q", strings.Split(",,", ",")))
	for _, n := range []int{-1, 0, 1, 2, 3, 10} {
		row(fmt.Sprintf("SplitN %d", n), fmt.Sprintf("%q", strings.SplitN("a,b,c,d", ",", n)))
	}
	row("SplitN 0 nil", strings.SplitN("a,b", ",", 0) == nil)
	row("SplitN empty sep 2", fmt.Sprintf("%q", strings.SplitN("abc", "", 2)))
	row("SplitAfter", fmt.Sprintf("%q", strings.SplitAfter("a,b,c", ",")))
	row("SplitAfter empty sep", fmt.Sprintf("%q", strings.SplitAfter("abc", "")))
	row("SplitAfterN 2", fmt.Sprintf("%q", strings.SplitAfterN("a,b,c", ",", 2)))
	row("SplitAfterN 0", strings.SplitAfterN("a,b", ",", 0) == nil)
	row("Fields", fmt.Sprintf("%q", strings.Fields("  foo bar  baz   ")))
	row("Fields tabs", fmt.Sprintf("%q", strings.Fields("\tfoo\nbar baz\vqux")))
	row("Fields empty", len(strings.Fields("   ")))
	row("FieldsFunc", fmt.Sprintf("%q", strings.FieldsFunc("a1b22c333", unicode.IsDigit)))
	row("FieldsFunc none", len(strings.FieldsFunc("", unicode.IsDigit)))
	row("Join", strings.Join([]string{"a", "b", "c"}, "-"))
	row("Join 1", strings.Join([]string{"a"}, "-"))
	row("Join none", fmt.Sprintf("%q", strings.Join(nil, "-")))
	row("Join empty sep", strings.Join([]string{"a", "b"}, ""))
	b, a, ok := strings.Cut("key=value=more", "=")
	row("Cut", fmt.Sprintf("%q %q %v", b, a, ok))
	b, a, ok = strings.Cut("novalue", "=")
	row("Cut absent", fmt.Sprintf("%q %q %v", b, a, ok))
	b, a, ok = strings.Cut("=", "=")
	row("Cut only sep", fmt.Sprintf("%q %q %v", b, a, ok))
	b, a, ok = strings.CutLast("key=value=more", "=")
	row("CutLast", fmt.Sprintf("%q %q %v", b, a, ok))
	b, a, ok = strings.CutLast("plain", "=")
	row("CutLast absent", fmt.Sprintf("%q %q %v", b, a, ok))
	a, ok = strings.CutPrefix("prefix-body", "prefix-")
	row("CutPrefix", fmt.Sprintf("%q %v", a, ok))
	a, ok = strings.CutPrefix("body", "prefix-")
	row("CutPrefix absent", fmt.Sprintf("%q %v", a, ok))
	a, ok = strings.CutPrefix("body", "")
	row("CutPrefix empty", fmt.Sprintf("%q %v", a, ok))
	b, ok = strings.CutSuffix("body.txt", ".txt")
	row("CutSuffix", fmt.Sprintf("%q %v", b, ok))
	b, ok = strings.CutSuffix("body", ".txt")
	row("CutSuffix absent", fmt.Sprintf("%q %v", b, ok))
}

func trims() {
	rowq("TrimSpace", strings.TrimSpace(" \t\n hello world \r\n"))
	rowq("TrimSpace only space", strings.TrimSpace("   "))
	rowq("Trim", strings.Trim("xxhixyx", "xy"))
	rowq("Trim all", strings.Trim("xxx", "x"))
	rowq("Trim empty cutset", strings.Trim("  x ", ""))
	rowq("Trim rune", strings.Trim("日本語本日", "日本"))
	rowq("Trim invalid", strings.Trim("\xff語\xff", "\xff"))
	rowq("TrimLeftFunc invalid", strings.TrimLeftFunc("\xffabc", func(r rune) bool { return r == unicode.ReplacementChar }))
	rowq("TrimRightFunc invalid", strings.TrimRightFunc("abc\xff", func(r rune) bool { return r == unicode.ReplacementChar }))
	rowq("TrimLeft", strings.TrimLeft("xxhixx", "x"))
	rowq("TrimRight", strings.TrimRight("xxhixx", "x"))
	rowq("TrimPrefix", strings.TrimPrefix("prefix-body", "prefix-"))
	rowq("TrimPrefix absent", strings.TrimPrefix("body", "prefix-"))
	rowq("TrimSuffix", strings.TrimSuffix("body.txt", ".txt"))
	rowq("TrimSuffix absent", strings.TrimSuffix("body", ".txt"))
	rowq("TrimFunc", strings.TrimFunc("123abc456", unicode.IsDigit))
	rowq("TrimFunc all", strings.TrimFunc("123", unicode.IsDigit))
	rowq("TrimLeftFunc", strings.TrimLeftFunc("123abc456", unicode.IsDigit))
	rowq("TrimRightFunc", strings.TrimRightFunc("123abc456", unicode.IsDigit))
}

func cases() {
	rowq("ToUpper", strings.ToUpper("Hello, World! 123"))
	rowq("ToLower", strings.ToLower("Hello, World! 123"))
	rowq("ToUpper sharp s", strings.ToUpper("straße"))
	rowq("ToUpper dz digraph", strings.ToUpper("ǅǆ"))
	rowq("ToLower dz digraph", strings.ToLower("Ǆǅ"))
	rowq("ToTitle dz digraph", strings.ToTitle("Ǆǆ"))
	rowq("ToUpper long s", strings.ToUpper("ſ"))
	rowq("ToLower kelvin", strings.ToLower("K"))
	rowq("ToLower dotted I", strings.ToLower("İ"))
	rowq("ToUpper dotless i", strings.ToUpper("ı"))
	rowq("ToLower sigma", strings.ToLower("ΣΑΣ"))
	rowq("ToUpper greek", strings.ToUpper("αβγ"))
	rowq("ToUpper invalid", strings.ToUpper("a\xffb"))
	rowq("ToLower invalid", strings.ToLower("A\xffB"))
	rowq("ToUpper ascii fast", strings.ToUpper("already UPPER"))
	rowq("ToLowerSpecial turkish", strings.ToLowerSpecial(unicode.TurkishCase, "İI"))
	rowq("ToUpperSpecial turkish", strings.ToUpperSpecial(unicode.TurkishCase, "iı"))
	rowq("ToTitleSpecial turkish", strings.ToTitleSpecial(unicode.TurkishCase, "i"))
	rowq("ToTitle", strings.ToTitle("hello ǳ"))
	rowq("Title", strings.Title("hello wORLD o'neil x-ray 3d"))
	rowq("Title unicode", strings.Title("élan vital"))
	row("EqualFold", strings.EqualFold("Go", "GO"))
	row("EqualFold sharp s", strings.EqualFold("ß", "SS"))
	row("EqualFold kelvin", strings.EqualFold("K", "k"))
	row("EqualFold long s", strings.EqualFold("ſ", "S"))
	row("EqualFold sigma", strings.EqualFold("σ", "ς"))
	row("EqualFold length", strings.EqualFold("ab", "abc"))
	row("EqualFold invalid", strings.EqualFold("a\xff", "A\xff"))
	rowq("Map", strings.Map(func(r rune) rune {
		if r == 'a' {
			return 'A'
		}
		if r == 'x' {
			return -1
		}
		return r
	}, "banana xylophone"))
	rowq("Map rot13", strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		}
		return r
	}, "Hello, World!"))
	rowq("Map invalid", strings.Map(func(r rune) rune { return r }, "a\xffb"))
	rowq("Map grow", strings.Map(func(r rune) rune { return '日' }, "abc"))
	rowq("ToValidUTF8", strings.ToValidUTF8("a\xffb\xfe\xfdc", "?"))
	rowq("ToValidUTF8 empty repl", strings.ToValidUTF8("a\xffb", ""))
	rowq("Repeat", strings.Repeat("ab", 3))
	rowq("Repeat 0", strings.Repeat("ab", 0))
	rowq("Repeat of empty", strings.Repeat("", 5))
	rowq("Repeat 1", strings.Repeat("xyz", 1))
	row("Repeat long len", len(strings.Repeat("abc", 1000)))
	rowq("Replace", strings.Replace("oink oink oink", "k", "ky", 2))
	rowq("Replace -1", strings.Replace("oink oink oink", "oink", "moo", -1))
	rowq("Replace 0", strings.Replace("oink", "o", "a", 0))
	rowq("Replace empty old", strings.Replace("abc", "", "-", -1))
	rowq("Replace empty old 2", strings.Replace("abc", "", "-", 2))
	rowq("Replace empty old rune", strings.Replace("日本", "", "-", -1))
	rowq("ReplaceAll", strings.ReplaceAll("a.b.c", ".", "::"))
	rowq("ReplaceAll absent", strings.ReplaceAll("abc", "x", "y"))
	rowq("Clone", strings.Clone("clone"))
}

func replacer() {
	r := strings.NewReplacer("a", "1", "b", "2")
	rowq("Replacer bytes", r.Replace("abcab"))
	r = strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;")
	rowq("Replacer html", r.Replace("<a href=x&y>"))
	r = strings.NewReplacer("aaa", "3", "aa", "2", "a", "1")
	rowq("Replacer overlap long first", r.Replace("aaaaaa a aa"))
	r = strings.NewReplacer("a", "1", "aa", "2", "aaa", "3")
	rowq("Replacer overlap short first", r.Replace("aaaaaa a aa"))
	r = strings.NewReplacer("ab", "X", "bc", "Y")
	rowq("Replacer overlap pairs", r.Replace("abc bc ab"))
	r = strings.NewReplacer("", "-")
	rowq("Replacer empty old", r.Replace("abc"))
	r = strings.NewReplacer("", "-", "b", "B")
	rowq("Replacer empty and byte", r.Replace("abc"))
	r = strings.NewReplacer("hello", "bye", "hell", "heck")
	rowq("Replacer prefix pair", r.Replace("hello hell"))
	r = strings.NewReplacer("日", "day", "本", "book")
	rowq("Replacer runes", r.Replace("日本語"))
	r = strings.NewReplacer()
	rowq("Replacer none", r.Replace("same"))
	r = strings.NewReplacer("x", "")
	rowq("Replacer delete", r.Replace("xaxbx"))
	var sb strings.Builder
	n, err := strings.NewReplacer("a", "b").WriteString(&sb, "banana")
	row("Replacer WriteString", fmt.Sprintf("%d %v %q", n, err, sb.String()))
	r = strings.NewReplacer("a", "b", "b", "a")
	rowq("Replacer swap", r.Replace("abba"))
}

func builder() {
	var b strings.Builder
	row("Builder zero Len", b.Len())
	rowq("Builder zero String", b.String())
	n, err := b.WriteString("hello")
	row("WriteString", fmt.Sprintf("%d %v", n, err))
	err = b.WriteByte(' ')
	row("WriteByte", err)
	n, err = b.WriteRune('日')
	row("WriteRune", fmt.Sprintf("%d %v", n, err))
	n, err = b.WriteRune(-1)
	row("WriteRune invalid", fmt.Sprintf("%d %v", n, err))
	n, err = b.Write([]byte("!?"))
	row("Write", fmt.Sprintf("%d %v", n, err))
	rowq("String", b.String())
	row("Len", b.Len())
	fmt.Fprintf(&b, " %d-%s", 42, "x")
	rowq("Fprintf into Builder", b.String())
	b.Reset()
	row("Len after Reset", b.Len())
	rowq("String after Reset", b.String())
	for i := 0; i < 5; i++ {
		b.WriteString("ab")
	}
	rowq("loop", b.String())
	var w io.Writer = &b
	io.WriteString(w, "-end")
	rowq("as io.Writer", b.String())
}

func reader() {
	r := strings.NewReader("Hello, 世界! bye")
	row("Len", r.Len())
	row("Size", r.Size())
	buf := make([]byte, 5)
	n, err := r.Read(buf)
	row("Read", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	row("Len after Read", r.Len())
	c, err := r.ReadByte()
	row("ReadByte", fmt.Sprintf("%q %v", c, err))
	err = r.UnreadByte()
	row("UnreadByte", err)
	c, _ = r.ReadByte()
	row("ReadByte again", fmt.Sprintf("%q", c))
	r.ReadByte()
	ch, size, err := r.ReadRune()
	row("ReadRune", fmt.Sprintf("%q %d %v", ch, size, err))
	err = r.UnreadRune()
	row("UnreadRune", err)
	ch, size, _ = r.ReadRune()
	row("ReadRune again", fmt.Sprintf("%q %d", ch, size))
	err = r.UnreadByte()
	row("UnreadByte after rune", err)
	pos, err := r.Seek(0, io.SeekStart)
	row("Seek start", fmt.Sprintf("%d %v", pos, err))
	pos, err = r.Seek(-3, io.SeekEnd)
	row("Seek end", fmt.Sprintf("%d %v", pos, err))
	rest, _ := io.ReadAll(r)
	row("rest", fmt.Sprintf("%q", rest))
	pos, err = r.Seek(2, io.SeekCurrent)
	row("Seek past end", fmt.Sprintf("%d %v", pos, err))
	n, err = r.Read(buf)
	row("Read at end", fmt.Sprintf("%d %v", n, err))
	_, err = r.Seek(-1, io.SeekStart)
	row("Seek negative", err)
	n, err = r.ReadAt(buf, 7)
	row("ReadAt", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = r.ReadAt(buf, 16)
	row("ReadAt short", fmt.Sprintf("%d %v %q", n, err, buf[:n]))
	n, err = r.ReadAt(buf, -1)
	row("ReadAt negative", fmt.Sprintf("%d %v", n, err))
	r.Reset("new text")
	row("Reset Len", r.Len())
	var sb strings.Builder
	m, err := r.WriteTo(&sb)
	row("WriteTo", fmt.Sprintf("%d %v %q", m, err, sb.String()))
	row("Len after WriteTo", r.Len())
	_, err = r.ReadByte()
	row("ReadByte at end", err)
	_, _, err = r.ReadRune()
	row("ReadRune at end", err)
	empty := strings.NewReader("")
	n, err = empty.Read(buf)
	row("Read of empty", fmt.Sprintf("%d %v", n, err))
	n, err = strings.NewReader("abc").Read(nil)
	row("Read into nil", fmt.Sprintf("%d %v", n, err))
}

func iterators() {
	row("Lines", fmt.Sprintf("%q", slices.Collect(strings.Lines("a\nb\n\nc"))))
	row("Lines trailing", fmt.Sprintf("%q", slices.Collect(strings.Lines("a\nb\n"))))
	row("Lines empty", len(slices.Collect(strings.Lines(""))))
	row("Lines crlf", fmt.Sprintf("%q", slices.Collect(strings.Lines("a\r\nb"))))
	row("SplitSeq", fmt.Sprintf("%q", slices.Collect(strings.SplitSeq("a,b,,c", ","))))
	row("SplitSeq empty sep", fmt.Sprintf("%q", slices.Collect(strings.SplitSeq("日本", ""))))
	row("SplitSeq no match", fmt.Sprintf("%q", slices.Collect(strings.SplitSeq("abc", ","))))
	row("SplitSeq of empty", fmt.Sprintf("%q", slices.Collect(strings.SplitSeq("", ","))))
	row("SplitAfterSeq", fmt.Sprintf("%q", slices.Collect(strings.SplitAfterSeq("a,b,c", ","))))
	row("FieldsSeq", fmt.Sprintf("%q", slices.Collect(strings.FieldsSeq("  a  b\tc\n"))))
	row("FieldsSeq empty", len(slices.Collect(strings.FieldsSeq(" \t "))))
	row("FieldsFuncSeq", fmt.Sprintf("%q", slices.Collect(strings.FieldsFuncSeq("a1b2c", unicode.IsDigit))))
	count := 0
	for part := range strings.SplitSeq("a,b,c,d", ",") {
		count++
		if part == "b" {
			break
		}
	}
	row("SplitSeq break", count)
	var out []string
	for line := range strings.Lines("x\ny\nz\n") {
		out = append(out, strings.TrimSuffix(line, "\n"))
	}
	row("Lines loop", out)
}

func main() {
	fmt.Println("== the search functions ==")
	searches()
	fmt.Println("== the split functions ==")
	splits()
	fmt.Println("== the trim functions ==")
	trims()
	fmt.Println("== the case functions ==")
	cases()
	fmt.Println("== Replacer ==")
	replacer()
	fmt.Println("== Builder ==")
	builder()
	fmt.Println("== Reader ==")
	reader()
	fmt.Println("== the iterator functions ==")
	iterators()
}
