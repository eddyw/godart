package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main() {
	re := regexp.MustCompile(`a(x*)b`)
	fmt.Println(re.MatchString("-axxb-"), re.MatchString("-ab-"), re.MatchString("-cb-"))
	fmt.Println(re.FindString("-axxb-"), re.FindStringIndex("-axxb-"))
	fmt.Println(re.FindStringSubmatch("-axxb-"))
	fmt.Println(re.FindAllString("-ab-axb-axxb-", -1))
	fmt.Println(re.FindAllStringSubmatch("-ab-axb-", -1))
	fmt.Println(re.ReplaceAllString("-ab-axxb-", "T"))
	fmt.Println(re.ReplaceAllString("-axxb-", "[$1]"))
	fmt.Println(re.Split("1ab2axxb3", -1))
	fmt.Println(re.String(), re.NumSubexp())

	named := regexp.MustCompile(`(?P<key>\w+)=(?P<value>\d+)`)
	m := named.FindStringSubmatch("size=42")
	fmt.Println(m, named.SubexpNames())
	fmt.Println(named.ReplaceAllString("size=42", "${value}:${key}"))

	fold := regexp.MustCompile(`(?i)hello`)
	fmt.Println(fold.MatchString("HeLLo"), fold.MatchString("bye"))

	anchor := regexp.MustCompile(`^\d{3}-\d{4}$`)
	fmt.Println(anchor.MatchString("555-1234"), anchor.MatchString("x555-1234"))

	class := regexp.MustCompile(`[[:alpha:]]+|\p{Greek}+`)
	fmt.Println(class.FindAllString("abc ΩΨ 123 def", -1))

	fmt.Println(regexp.QuoteMeta("1+1=2?"))

	if _, err := regexp.Compile("a(b"); err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(regexp.MatchString(`\bgo\b`, "the go language"))

	b := regexp.MustCompile(`\d+`)
	fmt.Println(string(b.Find([]byte("abc 123 def"))), b.Match([]byte("none")))
	fmt.Println(string(b.ReplaceAll([]byte("a1b22c"), []byte("#"))))
	fmt.Println(b.FindAllStringIndex("a1b22c", -1))

	fmt.Println(re.ReplaceAllStringFunc("-ab-axxb-", strings.ToUpper))

	longest := regexp.MustCompile(`a|ab`)
	longest.Longest()
	fmt.Println(longest.FindString("abc"))

	fmt.Println(regexp.MustCompile(`(a)(b)?`).FindStringSubmatch("a"))
	fmt.Println(regexp.MustCompile(`x*`).FindAllString("abc", -1))
	fmt.Println(regexp.MustCompile(`(?s)a.b`).MatchString("a\nb"))
	fmt.Println(regexp.MustCompile(`(?m)^b`).FindAllString("a\nb\nb", -1))
}
