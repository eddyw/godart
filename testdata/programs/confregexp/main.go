// The program proves the package regexp against real Go: each find function
// with a limit and with a named group, each replace function with a group
// reference, Split, Longest, LiteralPrefix, QuoteMeta, the String form, the
// text of each compile error, and the Unicode classes.
package main

import (
	"fmt"
	"regexp"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

func finds() {
	re := regexp.MustCompile(`a(b*)c`)
	text := "xacybbcabcabbbc"
	row("String", re.String())
	row("MatchString", re.MatchString(text))
	row("MatchString absent", re.MatchString("xyz"))
	row("Match bytes", re.Match([]byte(text)))
	row("FindString", fmt.Sprintf("%q", re.FindString(text)))
	row("FindString absent", fmt.Sprintf("%q", re.FindString("zzz")))
	row("FindStringIndex", re.FindStringIndex(text))
	row("FindStringIndex absent", re.FindStringIndex("zzz") == nil)
	row("FindIndex bytes", re.FindIndex([]byte(text)))
	row("Find bytes", fmt.Sprintf("%q", re.Find([]byte(text))))
	row("Find absent nil", re.Find([]byte("zzz")) == nil)
	for _, n := range []int{-1, 0, 1, 2, 10} {
		row(fmt.Sprintf("FindAllString %d", n), fmt.Sprintf("%q", re.FindAllString(text, n)))
	}
	row("FindAllString absent nil", re.FindAllString("zzz", -1) == nil)
	row("FindAllStringIndex", re.FindAllStringIndex(text, -1))
	row("FindAllStringIndex 2", re.FindAllStringIndex(text, 2))
	row("FindAll bytes", fmt.Sprintf("%q", re.FindAll([]byte(text), -1)))
	row("FindAllIndex", re.FindAllIndex([]byte(text), -1))
	row("FindStringSubmatch", fmt.Sprintf("%q", re.FindStringSubmatch(text)))
	row("FindStringSubmatchIndex", re.FindStringSubmatchIndex(text))
	row("FindSubmatch", fmt.Sprintf("%q", re.FindSubmatch([]byte(text))))
	row("FindSubmatchIndex", re.FindSubmatchIndex([]byte(text)))
	row("FindAllStringSubmatch", fmt.Sprintf("%q", re.FindAllStringSubmatch(text, -1)))
	row("FindAllStringSubmatch 2", fmt.Sprintf("%q", re.FindAllStringSubmatch(text, 2)))
	row("FindAllStringSubmatchIndex", re.FindAllStringSubmatchIndex(text, -1))
	row("FindAllSubmatch", fmt.Sprintf("%q", re.FindAllSubmatch([]byte(text), 2)))
	row("FindAllSubmatchIndex", re.FindAllSubmatchIndex([]byte(text), -1))
	opt := regexp.MustCompile(`(a)(x)?(b)`)
	row("unmatched group", fmt.Sprintf("%q", opt.FindStringSubmatch("ab")))
	row("unmatched group index", opt.FindStringSubmatchIndex("ab"))
	empty := regexp.MustCompile(`a*`)
	row("empty matches", fmt.Sprintf("%q", empty.FindAllString("baaab", -1)))
	row("empty match index", empty.FindAllStringIndex("baaab", -1))
	row("empty on empty", fmt.Sprintf("%q", empty.FindAllString("", -1)))
	row("NumSubexp", opt.NumSubexp())
	named := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})(?:-(?P<day>\d{2}))?`)
	m := named.FindStringSubmatch("date: 2024-03 and 2025-12-31")
	row("named FindStringSubmatch", fmt.Sprintf("%q", m))
	row("SubexpNames", fmt.Sprintf("%q", named.SubexpNames()))
	row("NumSubexp named", named.NumSubexp())
	for _, name := range []string{"year", "month", "day", "hour", ""} {
		row(fmt.Sprintf("SubexpIndex %q", name), named.SubexpIndex(name))
	}
	for i, name := range named.SubexpNames() {
		if name != "" {
			row("group "+name, fmt.Sprintf("%q", m[i]))
		}
	}
	all := named.FindAllStringSubmatch("2024-03, 2025-12-31, 1999-01-01", -1)
	for _, m := range all {
		row("match "+m[0], fmt.Sprintf("y=%s m=%s d=%q", m[1], m[2], m[3]))
	}
	angle := regexp.MustCompile(`(?P<key>\w+)=(?P<value>\w*)`)
	row("angle group", fmt.Sprintf("%q", angle.FindAllStringSubmatch("a=1 b= c=3", -1)))
	reader := regexp.MustCompile(`o+`)
	row("FindReaderIndex", reader.FindReaderIndex(strings.NewReader("foo boo")))
	row("FindReaderSubmatchIndex", reader.FindReaderSubmatchIndex(strings.NewReader("xoooy")))
	row("MatchReader", reader.MatchReader(strings.NewReader("no")))
	ok, err := regexp.MatchString(`^\d+$`, "12345")
	row("MatchString function", fmt.Sprint(ok, " ", err))
	ok, err = regexp.MatchString(`^\d+$`, "12a45")
	row("MatchString function false", fmt.Sprint(ok, " ", err))
	ok, err = regexp.MatchString(`(`, "x")
	row("MatchString bad pattern", fmt.Sprint(ok, " ", err))
	ok, err = regexp.Match(`b+`, []byte("abbc"))
	row("Match function", fmt.Sprint(ok, " ", err))
}

func replaces() {
	re := regexp.MustCompile(`(\w+)@(\w+)\.com`)
	text := "mail bob@example.com and eve@test.com now"
	row("ReplaceAllString", re.ReplaceAllString(text, "$2:$1"))
	row("ReplaceAllString braces", re.ReplaceAllString(text, "${1}_at_${2}"))
	row("ReplaceAllString $0", re.ReplaceAllString(text, "<$0>"))
	row("ReplaceAllString bad ref", re.ReplaceAllString(text, "$1x"))
	row("ReplaceAllString $$", re.ReplaceAllString(text, "$$1"))
	row("ReplaceAllString missing", re.ReplaceAllString(text, "$9"))
	row("ReplaceAllString empty", re.ReplaceAllString(text, ""))
	row("ReplaceAllLiteralString", re.ReplaceAllLiteralString(text, "$2:$1"))
	row("ReplaceAllStringFunc", re.ReplaceAllStringFunc(text, strings.ToUpper))
	row("ReplaceAll bytes", string(re.ReplaceAll([]byte(text), []byte("[$2]"))))
	row("ReplaceAllLiteral bytes", string(re.ReplaceAllLiteral([]byte(text), []byte("[$2]"))))
	row("ReplaceAllFunc bytes", string(re.ReplaceAllFunc([]byte(text), func(b []byte) []byte {
		return []byte(strings.Repeat("*", len(b)))
	})))
	named := regexp.MustCompile(`(?P<first>\w+) (?P<last>\w+)`)
	row("named reference", named.ReplaceAllString("Ada Lovelace", "${last}, ${first}"))
	row("named reference bare", named.ReplaceAllString("Ada Lovelace", "$last $first"))
	row("named reference unknown", named.ReplaceAllString("Ada Lovelace", "$middle."))
	empty := regexp.MustCompile(`a*`)
	row("empty pattern replace", empty.ReplaceAllString("baaac", "-"))
	row("empty pattern on empty", empty.ReplaceAllString("", "-"))
	row("anchor replace", regexp.MustCompile(`^`).ReplaceAllString("line", "> "))
	row("end anchor replace", regexp.MustCompile(`$`).ReplaceAllString("line", ";"))
	row("word boundary", regexp.MustCompile(`\b`).ReplaceAllString("ab cd", "|"))
	row("Expand", string(re.Expand(nil, []byte("$1 at $2"), []byte(text), re.FindSubmatchIndex([]byte(text)))))
	row("ExpandString", string(re.ExpandString([]byte("pre:"), "$2/$1", text, re.FindStringSubmatchIndex(text))))
	var out []byte
	for _, idx := range re.FindAllStringSubmatchIndex(text, -1) {
		out = re.ExpandString(out, "$1;", text, idx)
	}
	row("ExpandString loop", string(out))
	row("multi line replace", regexp.MustCompile(`(?m)^(\w)`).ReplaceAllString("ab\ncd\nef", "[$1]"))
	row("dot all", regexp.MustCompile(`(?s)a.b`).ReplaceAllString("a\nb a b", "X"))
	row("dot no all", regexp.MustCompile(`a.b`).ReplaceAllString("a\nb a b", "X"))
	row("case fold", regexp.MustCompile(`(?i)hello`).ReplaceAllString("Hello HELLO hello", "hi"))
	row("ungreedy", regexp.MustCompile(`(?U)a+`).FindString("aaa"))
	row("greedy", regexp.MustCompile(`a+`).FindString("aaa"))
	row("lazy quantifier", regexp.MustCompile(`<.+?>`).FindAllString("<a><b>", -1))
	row("greedy quantifier", regexp.MustCompile(`<.+>`).FindAllString("<a><b>", -1))
}

func splits() {
	re := regexp.MustCompile(`\s*,\s*`)
	for _, n := range []int{-1, 0, 1, 2, 3, 10} {
		row(fmt.Sprintf("Split %d", n), fmt.Sprintf("%q", re.Split("a , b,c ,, d", n)))
	}
	row("Split no match", fmt.Sprintf("%q", re.Split("abc", -1)))
	row("Split empty", fmt.Sprintf("%q", re.Split("", -1)))
	row("Split leading", fmt.Sprintf("%q", re.Split(",a,b,", -1)))
	row("Split empty pattern", fmt.Sprintf("%q", regexp.MustCompile(``).Split("abc", -1)))
	row("Split empty pattern 2", fmt.Sprintf("%q", regexp.MustCompile(``).Split("abc", 2)))
	row("Split a*", fmt.Sprintf("%q", regexp.MustCompile(`a*`).Split("baaac", -1)))
	row("Split z+", fmt.Sprintf("%q", regexp.MustCompile(`z+`).Split("pizza", -1)))
	row("Split on space", fmt.Sprintf("%q", regexp.MustCompile(` `).Split("a b  c", -1)))
}

func forms() {
	re := regexp.MustCompile(`a|ab`)
	row("leftmost first", re.FindString("ab"))
	re.Longest()
	row("Longest", re.FindString("ab"))
	posix := regexp.MustCompilePOSIX(`a|ab`)
	row("POSIX", posix.FindString("ab"))
	row("POSIX String", posix.String())
	alt := regexp.MustCompile(`(a|ab)(c|bcd)(d*)`)
	row("alternation first", fmt.Sprintf("%q", alt.FindStringSubmatch("abcd")))
	alt.Longest()
	row("alternation longest", fmt.Sprintf("%q", alt.FindStringSubmatch("abcd")))
	for _, p := range []string{`abc`, `abc.*`, `a+bc`, `^abc`, `abc$`, `(abc)def`, `a|b`, `ab(c|d)`, `\.x`, ``, `(?i)abc`, `日本.*`, `a\x00b`} {
		prefix, complete := regexp.MustCompile(p).LiteralPrefix()
		fmt.Printf("LiteralPrefix %-12q %-8q %v\n", p, prefix, complete)
	}
	for _, s := range []string{`a.b`, `[abc]`, `a+b*c?`, `(x|y)`, `$1`, `^$`, `\d`, `plain`, ``, `日本.語`, `{3}`, "a\nb", `a-b`, `#!`} {
		row(fmt.Sprintf("QuoteMeta %q", s), regexp.QuoteMeta(s))
	}
	row("QuoteMeta round trip", regexp.MustCompile(regexp.QuoteMeta(`a.b*c`)).MatchString("xa.b*cx"))
	row("QuoteMeta no match", regexp.MustCompile(regexp.QuoteMeta(`a.b*c`)).MatchString("aXbc"))
	for _, p := range []string{`a.b`, `(?i)x`, `[a-z]+`, `\d{2,3}`, `a|b`, `(?P<n>x)`, `\pL`, `[[:alpha:]]`, `(?s).`, `a{,2}`, `\Q.*\E`} {
		row(fmt.Sprintf("String %q", p), regexp.MustCompile(p).String())
	}
	copied := regexp.MustCompile(`x+`)
	row("MarshalText", func() string { b, err := copied.MarshalText(); return fmt.Sprintf("%q %v", b, err) }())
	var back regexp.Regexp
	err := back.UnmarshalText([]byte(`y+`))
	row("UnmarshalText", fmt.Sprintf("%v %q", err, back.String()))
	err = back.UnmarshalText([]byte(`(`))
	row("UnmarshalText bad", err)
}

func errs() {
	patterns := []string{`(`, `)`, `[`, `a**`, `a{2,1}`, `\8`, `(?P<n>a)(?P<n>b)`, `(?P<1>a)`, `*a`, `a{1001}`, `\pX`, `[z-a]`, `(?z)`, `a\`, `[[:foo:]]`, `x{2}{3}`, `\C`, `(?P<>a)`, `a{1000}{1000}`, `(?i`, `\Qab`, `(?<n>a)(?<n>b)`, `\z`, `\Z`, `(?P=n)`, `[\d-z]`, `\x{110000}`}
	for _, p := range patterns {
		_, err := regexp.Compile(p)
		fmt.Printf("Compile %-20q %v\n", p, err)
	}
	catch("MustCompile panic", func() { regexp.MustCompile(`(`) })
	catch("MustCompile quote", func() { regexp.MustCompile("a\nb(") })
	_, err := regexp.CompilePOSIX(`\d`)
	row("CompilePOSIX perl class", err)
	_, err = regexp.CompilePOSIX(`a{2}`)
	row("CompilePOSIX repeat", err)
	_, err = regexp.Compile(`(?<name>x)`)
	row("Compile angle name", err)
	_, err = regexp.Compile(`\Q.*\E+`)
	row("Compile Q E", err)
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-30s %v\n", label, recover())
	}()
	f()
}

func classes() {
	text := "Ab1 _-日本語 αβγ ½ ٣ ℕ 😀 ǅ\t\n\u00a0"
	classes := []string{`\pL`, `\p{Lu}`, `\p{Ll}`, `\p{Lt}`, `\pN`, `\p{Nd}`, `\p{No}`, `\p{Han}`, `\p{Greek}`, `\p{Latin}`, `\p{Arabic}`,
		`\pS`, `\p{So}`, `\p{Sm}`, `\pP`, `\p{Pc}`, `\p{Pd}`, `\pZ`, `\p{Zs}`, `\pC`, `\p{Cc}`, `\PL`, `\P{Han}`, `[\p{L}\p{N}]`, `[^\p{L}]`,
		`\w`, `\W`, `\d`, `\D`, `\s`, `\S`, `[[:alpha:]]`, `[[:digit:]]`, `[[:alnum:]]`, `[[:space:]]`, `[[:upper:]]`, `[[:lower:]]`, `[[:punct:]]`, `[[:word:]]`, `[[:xdigit:]]`, `[[:cntrl:]]`, `[[:graph:]]`, `[[:print:]]`, `[[:blank:]]`, `[[:ascii:]]`, `[[:^alpha:]]`,
		`.`, `(?s).`, `[^\x00-\x7f]`, `[\x{100}-\x{ffff}]`, `[\x{10000}-\x{10ffff}]`, `(?i)ǆ`, `(?i)k`, `(?i)σ`, `(?i)ß`}
	for _, c := range classes {
		re := regexp.MustCompile(c)
		matches := re.FindAllString(text, -1)
		fmt.Printf("class %-24s %d %q\n", c, len(matches), strings.Join(matches, ""))
	}
	row("rune count of the text", len([]rune(text)))
	fold := regexp.MustCompile(`(?i)straße`)
	row("fold sharp s", fold.MatchString("STRASSE"))
	row("fold sharp s upper", fold.MatchString("STRAßE"))
	row("kelvin fold", regexp.MustCompile(`(?i)k`).FindAllString("kKK", -1))
	row("invalid utf8 match", regexp.MustCompile(`.`).FindAllString("a\xffb", -1))
	row("invalid utf8 as class", regexp.MustCompile(`\x{FFFD}`).FindAllStringIndex("a\xffb", -1))
	row("byte match", regexp.MustCompile(`\xff`).MatchString("\xff"))
	row("escaped unicode", regexp.MustCompile(`\x{65E5}\x{672C}`).MatchString("日本"))
	row("range of runes", regexp.MustCompile(`[一-龥]+`).FindString("abc日本語def"))
	row("negated range", regexp.MustCompile(`[^一-龥a-z]+`).FindString("abc日本語DEF"))
	row("repeat runes", regexp.MustCompile(`日{2}`).MatchString("日日"))
	row("anchor with runes", regexp.MustCompile(`^日.語$`).MatchString("日本語"))
	row("word boundary unicode", regexp.MustCompile(`\b\w+\b`).FindAllString("日本 abc", -1))
	row("rune index", regexp.MustCompile(`本`).FindStringIndex("日本語"))
	row("Longest with fold", regexp.MustCompile(`(?i)a|ab`).FindString("AB"))
}

func main() {
	fmt.Println("== the find functions ==")
	finds()
	fmt.Println("== the replace functions ==")
	replaces()
	fmt.Println("== Split ==")
	splits()
	fmt.Println("== the forms ==")
	forms()
	fmt.Println("== the compile errors ==")
	errs()
	fmt.Println("== the classes ==")
	classes()
}
