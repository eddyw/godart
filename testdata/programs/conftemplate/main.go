// The program proves the package text/template against real Go: each action
// of the language, each comparison function, each escape function, a custom
// function map, each option of a missing key, a nil pointer, a map with
// sorted keys, and the text of each parse error and of each execution error.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type address struct {
	City string
	Zip  int
}

type person struct {
	Name    string
	Age     int
	Tags    []string
	Home    *address
	Scores  map[string]int
	private int
}

func (p person) Greeting() string { return "hi " + p.Name }

func (p person) Add(a, b int) int { return a + b }

func (p *person) Pointer() string { return "ptr " + p.Name }

func (p person) Fail() (string, error) { return "", errors.New("the method failed") }

func (p person) Maybe(ok bool) (string, error) {
	if !ok {
		return "", fmt.Errorf("maybe failed for %s", p.Name)
	}
	return "maybe ok", nil
}

var ada = person{
	Name:   "Ada",
	Age:    36,
	Tags:   []string{"math", "engine", "notes"},
	Home:   &address{City: "London", Zip: 1815},
	Scores: map[string]int{"z": 26, "a": 1, "m": 13},
}

var funcs = template.FuncMap{
	"upper":  strings.ToUpper,
	"repeat": strings.Repeat,
	"join":   strings.Join,
	"add":    func(a, b int) int { return a + b },
	"fail":   func() (string, error) { return "", errors.New("the function failed") },
	"pair":   func(a, b any) []any { return []any{a, b} },
	"seq": func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	},
}

// run parses and executes the text with the data, and it prints the result or
// the error under the label. A newline inside the output takes the form \n,
// thus each case stays on 1 line.
func run(label, text string, data any) {
	t, err := template.New(label).Funcs(funcs).Parse(text)
	if err != nil {
		fmt.Printf("%-28s parse error: %v\n", label, err)
		return
	}
	var sb strings.Builder
	err = t.Execute(&sb, data)
	out := strings.ReplaceAll(sb.String(), "\n", "\\n")
	if err != nil {
		fmt.Printf("%-28s %q exec error: %v\n", label, out, err)
		return
	}
	fmt.Printf("%-28s %q\n", label, out)
}

func actions() {
	run("text only", "plain text", nil)
	run("field", "{{.Name}} is {{.Age}}", ada)
	run("nested field", "{{.Home.City}} {{.Home.Zip}}", ada)
	run("dot", "{{.}}", 42)
	run("dot string", "{{.}}", "text")
	run("dot nil", "[{{.}}]", nil)
	run("method", "{{.Greeting}}", ada)
	run("method with args", "{{.Add 2 3}}", ada)
	run("pointer method", "{{.Pointer}}", &ada)
	run("pointer field", "{{.Name}}", &ada)
	run("method with error", "{{.Fail}}", ada)
	run("method error result", "{{.Maybe true}} {{.Maybe false}}", ada)
	run("pipeline", "{{.Name | upper}}", ada)
	run("pipeline chain", "{{.Name | upper | printf \"%q\"}}", ada)
	run("pipeline args", "{{\"ab\" | repeat 3}}", nil)
	run("parenthesis", "{{(index .Tags 1) | upper}}", ada)
	run("if true", "{{if .Age}}has age{{end}}", ada)
	run("if false", "{{if .private}}private{{else}}zero{{end}}", ada)
	run("if empty slice", "{{if .}}some{{else}}none{{end}}", []int{})
	run("if nil slice", "{{if .}}some{{else}}none{{end}}", []int(nil))
	run("if empty map", "{{if .}}some{{else}}none{{end}}", map[string]int{})
	run("if empty string", "{{if .}}some{{else}}none{{end}}", "")
	run("if false bool", "{{if .}}some{{else}}none{{end}}", false)
	run("if zero float", "{{if .}}some{{else}}none{{end}}", 0.0)
	run("if nil pointer", "{{if .}}some{{else}}none{{end}}", (*address)(nil))
	run("if struct", "{{if .}}some{{else}}none{{end}}", address{})
	run("else if", "{{if eq .Age 1}}one{{else if eq .Age 36}}thirty six{{else}}other{{end}}", ada)
	run("else if chain", "{{if lt . 10}}small{{else if lt . 100}}medium{{else if lt . 1000}}large{{else}}huge{{end}}", 500)
	run("range", "{{range .Tags}}[{{.}}]{{end}}", ada)
	run("range index", "{{range $i, $t := .Tags}}{{$i}}={{$t}} {{end}}", ada)
	run("range else", "{{range .}}x{{else}}empty{{end}}", []int{})
	run("range else nil", "{{range .}}x{{else}}empty{{end}}", nil)
	run("range map", "{{range $k, $v := .Scores}}{{$k}}:{{$v}} {{end}}", ada)
	run("range map keys", "{{range $k, $v := .}}{{$k}} {{end}}", map[int]string{3: "c", 1: "a", 2: "b"})
	run("range int", "{{range 4}}{{.}}{{end}}", nil)
	run("range int var", "{{range $i := 3}}<{{$i}}>{{end}}", nil)
	run("range break", "{{range .Tags}}{{if eq . \"engine\"}}{{break}}{{end}}{{.}} {{end}}", ada)
	run("range continue", "{{range .Tags}}{{if eq . \"engine\"}}{{continue}}{{end}}{{.}} {{end}}", ada)
	run("range dot outside", "{{range .Tags}}{{$.Name}}:{{.}} {{end}}", ada)
	run("range channel", "{{range .}}{{.}} {{end}}", makeChan())
	run("range seq", "{{range seq 3}}{{.}}{{end}}", nil)
	run("with", "{{with .Home}}{{.City}}{{end}}", ada)
	run("with nil", "{{with .Home}}{{.City}}{{else}}no home{{end}}", person{})
	run("with else with", "{{with .private}}p{{else with .Age}}age {{.}}{{else}}none{{end}}", ada)
	run("with dot outside", "{{with .Home}}{{$.Name}} in {{.City}}{{end}}", ada)
	run("variable", "{{$x := .Name}}{{$x}} {{$x | upper}}", ada)
	run("variable assign", "{{$x := 1}}{{$x = 2}}{{$x}}", nil)
	run("variable scope", "{{$x := 1}}{{if true}}{{$x = 5}}{{end}}{{$x}}", nil)
	run("variable in range", "{{$sum := 0}}{{range .}}{{$sum = add $sum .}}{{end}}{{$sum}}", []int{1, 2, 3})
	run("dollar root", "{{$.Name}}", ada)
	run("printf", "{{printf \"%s is %d and %v\" .Name .Age .Tags}}", ada)
	run("printf verbs", "{{printf \"%05.2f|%x|%q|%t|%c\" 3.14159 255 \"q\" true 65}}", nil)
	run("print", "{{print 1 2 \"a\" \"b\" 3}}", nil)
	run("println", "{{println 1 2 \"a\"}}", nil)
	run("len", "{{len .Tags}} {{len .Name}} {{len .Scores}}", ada)
	run("len of int", "{{len .}}", 5)
	run("index slice", "{{index .Tags 0}} {{index .Tags 2}}", ada)
	run("index out of range", "{{index .Tags 5}}", ada)
	run("index negative", "{{index .Tags -1}}", ada)
	run("index map", "{{index .Scores \"a\"}} {{index .Scores \"missing\"}}", ada)
	run("index nested", "{{index . 1 0}}", [][]string{{"a"}, {"b", "c"}})
	run("index string", "{{index . 1}}", "abc")
	run("index nil", "{{index . 0}}", nil)
	run("slice", "{{slice .Tags 1}} {{slice .Tags 0 2}} {{slice .Tags 1 2}}", ada)
	run("slice string", "{{slice . 1 3}}", "abcdef")
	run("slice 3 index", "{{slice . 0 1 2}}", []int{1, 2, 3})
	run("slice out of range", "{{slice . 2 1}}", []int{1, 2, 3})
	run("comment", "a{{/* a comment */}}b", nil)
	run("trim markers", "a  {{- \"x\" -}}  b", nil)
	run("trim left only", "a  {{- \"x\"}}  b", nil)
	run("trim newline", "a\n{{- \"x\" -}}\nb", nil)
	run("raw string", "{{`raw \"quoted\"`}}", nil)
	run("char constant", "{{'a'}} {{printf \"%c\" 'b'}}", nil)
	run("numbers", "{{1}} {{1.5}} {{1e3}} {{0x10}} {{-3}} {{1_000}}", nil)
	run("nil literal", "{{if eq nil nil}}same{{end}}", nil)
	run("bool literal", "{{true}} {{false}}", nil)
	run("call", "{{call .}}", func() string { return "called" })
	run("call with args", "{{call . 2 3}}", func(a, b int) int { return a * b })
	run("field of map", "{{.a}} {{.z}} [{{.missing}}]", map[string]int{"a": 1, "z": 26})
	run("field chain on map", "{{.inner.x}}", map[string]map[string]int{"inner": {"x": 7}})
	run("interface field", "{{.V}}", struct{ V any }{V: 3})
	run("interface nil field", "[{{.V}}]", struct{ V any }{})
	run("nil pointer field", "{{.Home.City}}", person{})
	run("nil pointer print", "[{{.Home}}]", person{})
	run("nil pointer if", "{{if .Home}}has{{else}}none{{end}}", person{})
	run("unexported field", "{{.private}}", ada)
	run("missing field", "{{.Missing}}", ada)
	run("missing method", "{{.Missing}}", &ada)
	run("field on int", "{{.Name}}", 5)
	run("field on nil", "{{.Name}}", nil)
	run("byte slice", "{{.}} {{printf \"%s\" .}}", []byte("bytes"))
	run("error value", "{{.}}", errors.New("an error value"))
	run("stringer", "{{.}}", stringer{})
	run("float", "{{.}} {{printf \"%.2f\" .}}", 2.5)
	run("negative", "{{.}}", -7)
	run("struct print", "{{.}}", address{"Paris", 75})
	run("struct pointer print", "{{.Home}}", ada)
}

type stringer struct{}

func (stringer) String() string { return "the stringer text" }

func makeChan() chan int {
	c := make(chan int, 3)
	c <- 7
	c <- 8
	c <- 9
	close(c)
	return c
}

func comparisons() {
	run("eq", "{{eq 1 1}} {{eq 1 2}} {{eq \"a\" \"a\"}} {{eq .Name \"Ada\"}}", ada)
	run("eq many", "{{eq 1 2 3 1}} {{eq 1 2 3}}", nil)
	run("eq mixed int", "{{eq .A .B}}", struct {
		A int
		B int64
	}{1, 1})
	run("eq int uint", "{{eq .A .B}}", struct {
		A int
		B uint
	}{1, 1})
	run("eq int float", "{{eq 1 1.0}}", nil)
	run("eq int string", "{{eq 1 \"1\"}}", nil)
	run("eq bool", "{{eq true true}} {{eq true false}}", nil)
	run("eq nil", "{{eq . nil}}", nil)
	run("eq nil pointer", "{{eq .Home nil}}", person{})
	run("eq struct", "{{eq . .}}", address{})
	run("ne", "{{ne 1 2}} {{ne \"a\" \"a\"}}", nil)
	run("lt", "{{lt 1 2}} {{lt 2 1}} {{lt \"a\" \"b\"}} {{lt 1.5 2.5}}", nil)
	run("le", "{{le 1 1}} {{le 2 1}}", nil)
	run("gt", "{{gt 1 2}} {{gt 2 1}} {{gt \"b\" \"a\"}}", nil)
	run("ge", "{{ge 1 1}} {{ge 1 2}}", nil)
	run("lt int uint", "{{lt .A .B}}", struct {
		A int
		B uint
	}{-1, 1})
	run("lt mixed", "{{lt 1 \"a\"}}", nil)
	run("lt bool", "{{lt true false}}", nil)
	run("and", "{{and 1 2}} {{and 0 2}} {{and 1 0}} {{and \"\" 1}} {{and true \"x\"}}", nil)
	run("and short", "{{and false .Missing}}", ada)
	run("or", "{{or 0 2}} {{or 0 0}} {{or \"\" \"x\"}} {{or 1 .Missing}}", ada)
	run("or many", "{{or 0 \"\" false 3 4}}", nil)
	run("not", "{{not true}} {{not 0}} {{not 1}} {{not \"\"}} {{not .Tags}}", ada)
	run("and or not", "{{if and (gt .Age 30) (not (eq .Name \"Bob\"))}}ok{{end}}", ada)
	run("and one arg", "{{and 5}} {{or 0}}", nil)
	run("html", "{{html \"<a href='x'>&\\\"q\\\"</a>\"}}", nil)
	run("html of value", "{{html .}}", "<b>bold</b>")
	run("html of int", "{{html 5}}", nil)
	run("html many", "{{html \"<\" 1 \">\"}}", nil)
	run("js", "{{js \"alert('x'); </script> \\\"q\\\" \\\\ \\n é\"}}", nil)
	run("js of value", "{{js .}}", "a+b=c&d")
	run("js of int", "{{js 5}}", nil)
	run("urlquery", "{{urlquery \"a b&c=d/e?f\"}} {{urlquery \"日本\"}}", nil)
	run("urlquery many", "{{urlquery \"a b\" \"c\"}}", nil)
	run("printf %v of nil", "{{printf \"%v\" nil}}", nil)
	run("printf error", "{{printf \"%d\" \"x\"}}", nil)
	run("printf too few", "{{printf \"%d %d\" 1}}", nil)
	run("printf type", "{{printf \"%T %T %T\" 1 \"a\" .}}", ada)
	run("custom upper", "{{upper \"go\"}}", nil)
	run("custom join", "{{join .Tags \", \"}}", ada)
	run("custom add", "{{add 2 3}} {{add (add 1 2) 3}}", nil)
	run("custom fail", "{{fail}}", nil)
	run("custom pair", "{{pair 1 \"a\"}}", nil)
	run("custom bad arg type", "{{add 1 \"a\"}}", nil)
	run("custom bad arg count", "{{add 1}}", nil)
	run("custom too many", "{{add 1 2 3}}", nil)
	run("unknown function", "{{unknown 1}}", nil)
}

func defines() {
	run("define and template", `{{define "item"}}<{{.}}>{{end}}{{range .Tags}}{{template "item" .}}{{end}}`, ada)
	run("template no data", `{{define "t"}}[{{.}}]{{end}}{{template "t"}}`, ada)
	run("template pipeline", `{{define "t"}}[{{.}}]{{end}}{{template "t" .Name | upper}}`, ada)
	run("define nested", `{{define "a"}}a({{template "b" .}}){{end}}{{define "b"}}b{{.}}{{end}}{{template "a" 1}}`, nil)
	run("template missing", `{{template "nothere" .}}`, ada)
	run("define only", `{{define "x"}}x{{end}}`, nil)
	run("define with space", `{{define "x"}}x{{end}}  `, nil)
	run("block", `{{block "b" .}}default {{.Name}}{{end}}`, ada)
	t := template.Must(template.New("root").Parse(`{{block "b" .}}default{{end}}`))
	template.Must(t.New("b").Parse(`override {{.Name}}`))
	var sb strings.Builder
	t.Execute(&sb, ada)
	fmt.Printf("%-28s %q\n", "block override", sb.String())
	t = template.Must(template.New("multi").Parse(`{{define "one"}}1{{end}}{{define "two"}}2{{end}}root`))
	names := []string{}
	for _, tt := range t.Templates() {
		names = append(names, tt.Name())
	}
	sortStrings(names)
	fmt.Printf("%-28s %v\n", "Templates", names)
	fmt.Printf("%-28s %v\n", "Lookup found", t.Lookup("one") != nil)
	fmt.Printf("%-28s %v\n", "Lookup missing", t.Lookup("three") == nil)
	fmt.Printf("%-28s %v\n", "DefinedTemplates", t.DefinedTemplates() != "")
	sb.Reset()
	err := t.ExecuteTemplate(&sb, "two", nil)
	fmt.Printf("%-28s %q %v\n", "ExecuteTemplate", sb.String(), err)
	err = t.ExecuteTemplate(&sb, "nine", nil)
	fmt.Printf("%-28s %v\n", "ExecuteTemplate missing", err)
	fmt.Printf("%-28s %v\n", "Name", t.Name())
	clone, err := t.Clone()
	fmt.Printf("%-28s %v %v\n", "Clone", clone.Name(), err)
	_, err = t.New("one").Parse("redefined")
	fmt.Printf("%-28s %v\n", "redefine after parse", err)
	sb.Reset()
	t.ExecuteTemplate(&sb, "one", nil)
	fmt.Printf("%-28s %q\n", "redefined text", sb.String())
	added, err := template.New("a").Parse(`{{define "shared"}}from a{{end}}`)
	fmt.Printf("%-28s %v\n", "AddParseTree source", err)
	other := template.New("b")
	_, err = other.AddParseTree("shared", added.Lookup("shared").Tree)
	sb.Reset()
	other.ExecuteTemplate(&sb, "shared", nil)
	fmt.Printf("%-28s %q %v\n", "AddParseTree", sb.String(), err)
	empty := template.New("empty")
	err = empty.Execute(&sb, nil)
	fmt.Printf("%-28s %v\n", "Execute with no parse", err)
	run("delims default", "{{.Name}} [[.Name]]", ada)
	d := template.Must(template.New("d").Delims("[[", "]]").Parse("{{.Name}} [[.Name]]"))
	sb.Reset()
	d.Execute(&sb, ada)
	fmt.Printf("%-28s %q\n", "Delims", sb.String())
	fmt.Printf("%-28s %v\n", "HTMLEscapeString", template.HTMLEscapeString("<a>&'\"\x00"))
	fmt.Printf("%-28s %v\n", "JSEscapeString", template.JSEscapeString("<a>&'\"\\\n\r\t\u2028é"))
	fmt.Printf("%-28s %v\n", "URLQueryEscaper", template.URLQueryEscaper("a b", 1, "&"))
	fmt.Printf("%-28s %v\n", "HTMLEscaper", template.HTMLEscaper("<", 2))
	fmt.Printf("%-28s %v\n", "JSEscaper", template.JSEscaper("'", 3))
	fmt.Printf("%-28s %v\n", "IsTrue slice", func() string { t, ok := template.IsTrue([]int{}); return fmt.Sprint(t, ok) }())
	fmt.Printf("%-28s %v\n", "IsTrue int", func() string { t, ok := template.IsTrue(3); return fmt.Sprint(t, ok) }())
	fmt.Printf("%-28s %v\n", "IsTrue nil", func() string { t, ok := template.IsTrue(nil); return fmt.Sprint(t, ok) }())
	fmt.Printf("%-28s %v\n", "IsTrue struct", func() string { t, ok := template.IsTrue(address{}); return fmt.Sprint(t, ok) }())
	catch("Must panic", func() { template.Must(template.New("bad").Parse("{{")) })
}

func sortStrings(list []string) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j] < list[j-1]; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-28s %v\n", label, recover())
	}()
	f()
}

func options() {
	data := map[string]any{"present": 1, "nilval": nil}
	for _, opt := range []string{"missingkey=default", "missingkey=invalid", "missingkey=zero", "missingkey=error"} {
		t := template.Must(template.New("opt").Option(opt).Parse("[{{.present}}] [{{.missing}}] [{{.nilval}}]"))
		var sb strings.Builder
		err := t.Execute(&sb, data)
		fmt.Printf("%-28s %q %v\n", opt, sb.String(), err)
		t = template.Must(template.New("opt2").Option(opt).Parse("[{{.missing}}]"))
		sb.Reset()
		err = t.Execute(&sb, map[string]int{"a": 1})
		fmt.Printf("%-28s %q %v\n", opt+" typed", sb.String(), err)
		t = template.Must(template.New("opt3").Option(opt).Parse("[{{if .missing}}yes{{else}}no{{end}}]"))
		sb.Reset()
		err = t.Execute(&sb, data)
		fmt.Printf("%-28s %q %v\n", opt+" in if", sb.String(), err)
	}
	catch("bad option", func() { template.New("x").Option("missingkey=bogus") })
	catch("empty option", func() { template.New("x").Option("") })
}

func errs() {
	bad := []string{"{{", "{{.Name", "{{end}}", "{{if}}", "{{if .}}", "{{range}}x{{end}}", "{{else}}", "{{if .}}a{{else}}b{{else}}c{{end}}",
		"{{template}}", "{{template .}}", "{{define}}x{{end}}", "{{$x}}", "{{$x := 1}}{{$x.y := 2}}", "{{.Name .Age}}", "{{1 2}}",
		"{{\"unterminated}}", "{{'ab'}}", "{{.Name | }}", "{{| .Name}}", "{{break}}", "{{continue}}", "{{range .}}{{end}}{{break}}",
		"{{with}}{{end}}", "{{nil}}", "{{.Name | nil}}", "{{(}}", "{{)}}", "{{/* unclosed", "{{- 1 -}}x{{-}}", "{{$x = 1}}",
		"{{range $a, $b, $c := .}}{{end}}", "{{define \"x\"}}", "{{block \"x\"}}{{end}}", "{{define \"a\"}}{{end}}{{define \"a\"}}{{end}}",
		"{{ .Name }}\n{{ end }}", "{{index}}", "{{len}}", "{{.Name}}{{else}}", "{{0x}}", "{{1e}}", "{{.1x}}", "{{if .}}{{else if}}{{end}}",
		"{{`raw}}", "{{if .}}{{else}}{{else if .}}{{end}}", "{{$}}", "{{$.}}", "{{..}}", "{{.Name.}}", "{{#}}", "{{@}}"}
	for _, text := range bad {
		_, err := template.New("bad").Funcs(funcs).Parse(text)
		fmt.Printf("parse %-48q %v\n", text, err)
	}
	execs := []struct {
		text string
		data any
	}{
		{"{{.Missing}}", ada}, {"{{.Name.Missing}}", ada}, {"{{index .Tags 9}}", ada}, {"{{index . \"k\"}}", []int{1}},
		{"{{.Home.City}}", person{}}, {"{{call .}}", 5}, {"{{call . 1}}", func() {}}, {"{{template \"none\"}}", nil},
		{"{{len .}}", 5}, {"{{slice . 5}}", []int{1}}, {"{{lt 1 \"a\"}}", nil}, {"{{eq . 1}}", []int{}}, {"{{.Fail}}", ada},
		{"{{fail}}", nil}, {"{{range .}}{{end}}", 5}, {"{{with .Home}}{{.Zip.X}}{{end}}", ada}, {"{{.Add 1}}", ada}, {"{{.Add 1 2 3}}", ada},
		{"{{.Add \"a\" 1}}", ada}, {"{{printf \"%d\" .}}", "x"}, {"{{index . 0}}", map[string]int{}}, {"{{.x}}", map[int]int{}},
		{"{{index . 1.5}}", []int{1}}, {"{{slice . 0 1}}", 5}, {"{{and .Missing 1}}", map[string]int{}}, {"{{.Name}}", []int{}},
		{"{{$x := .Missing}}{{$x}}", map[string]int{}}, {"{{not 1 2}}", nil},
		{"{{html}}", nil}, {"{{urlquery}}", nil}, {"{{.Greeting 1}}", ada}, {"{{.Pointer}}", ada}, {"{{index .Tags \"a\"}}", ada},
	}
	for _, c := range execs {
		t := template.Must(template.New("exec").Funcs(funcs).Parse(c.text))
		var sb strings.Builder
		err := t.Execute(&sb, c.data)
		fmt.Printf("exec %-36q %q %v\n", c.text, sb.String(), err)
	}
	var ee template.ExecError
	err := template.Must(template.New("named").Parse("{{.Missing}}")).Execute(&strings.Builder{}, ada)
	fmt.Printf("%-28s %v %v\n", "ExecError As", errors.As(err, &ee), ee.Name)
	fmt.Printf("%-28s %v\n", "ExecError Unwrap", errors.Unwrap(err) != nil)
	err = template.Must(template.New("wrapped").Parse("{{.Fail}}")).Execute(&strings.Builder{}, ada)
	fmt.Printf("%-28s %v %v\n", "method error As", errors.As(err, &ee), errors.Unwrap(err))
	failing := template.Must(template.New("w").Parse("text {{.}}"))
	err = failing.Execute(failWriter{}, 1)
	fmt.Printf("%-28s %v\n", "writer error", err)
	fmt.Printf("%-28s %v\n", "writer error As", errors.As(err, &ee))
	err = template.Must(template.New("os").Parse("Execute to Stdout            ok\n")).Execute(os.Stdout, nil)
	fmt.Printf("%-28s %v\n", "Execute to Stdout err", err)
}

type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, errors.New("the writer is closed") }

func main() {
	fmt.Println("== the actions ==")
	actions()
	fmt.Println("== the comparisons and the functions ==")
	comparisons()
	fmt.Println("== define and template ==")
	defines()
	fmt.Println("== the options ==")
	options()
	fmt.Println("== the errors ==")
	errs()
}
