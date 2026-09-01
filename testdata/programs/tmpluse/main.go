package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

type addr struct {
	City string
	Zip  string
}

type person struct {
	Name    string
	Age     int
	Tags    []string
	Home    *addr
	Scores  map[string]int
	private int
}

func (p person) Greet() string { return "hi " + p.Name }

func (p *person) Loud() string { return strings.ToUpper(p.Name) }

func run(name, text string, data any) {
	t, err := template.New(name).Funcs(template.FuncMap{
		"twice": func(s string) string { return s + s },
		"sum":   func(a, b int) int { return a + b },
	}).Parse(text)
	if err != nil {
		fmt.Println(name, "parse:", err)
		return
	}
	var out strings.Builder
	if err := t.Execute(&out, data); err != nil {
		fmt.Println(name, "exec:", err)
		return
	}
	fmt.Printf("%s: %s\n", name, out.String())
}

func main() {
	p := person{
		Name:   "ada",
		Age:    36,
		Tags:   []string{"math", "code"},
		Home:   &addr{City: "london", Zip: "N1"},
		Scores: map[string]int{"a": 1, "b": 2},
	}
	_ = p.private

	run("field", "{{.Name}} is {{.Age}}", p)
	run("nested", "{{.Home.City}} {{.Home.Zip}}", p)
	run("range", "{{range .Tags}}<{{.}}>{{end}}", p)
	run("index", "{{index .Tags 1}} {{index .Scores \"b\"}}", p)
	run("rangemap", "{{range $k, $v := .Scores}}{{$k}}={{$v}} {{end}}", p)
	run("if", "{{if gt .Age 30}}old{{else}}young{{end}}", p)
	run("with", "{{with .Home}}{{.City}}{{end}}", p)
	run("method", "{{.Greet}}", p)
	run("ptrmethod", "{{.Loud}}", &p)
	run("func", "{{twice .Name}} {{sum .Age 4}}", p)
	run("pipe", "{{.Name | twice | printf \"%q\"}}", p)
	run("var", "{{$n := .Name}}{{$n}}", p)
	run("builtin", "{{len .Tags}} {{printf \"%d\" .Age}} {{js .Name}}", p)
	run("compare", "{{eq .Name \"ada\"}} {{ne .Age 1}} {{and true .Age}} {{or false .Name}}", p)
	run("slice", "{{slice .Tags 0 1}}", p)
	run("html", "{{. | html}}", "<b>&</b>")
	run("nil", "{{if .Home}}yes{{end}}", person{})
	run("template", `{{define "x"}}[{{.}}]{{end}}{{template "x" .Name}}`, p)
	run("comment", "a{{/* gone */}}b", nil)
	run("dollar", "{{range .Tags}}{{$.Name}}:{{.}} {{end}}", p)

	// An error of the execution reaches the caller.
	run("bad", "{{.Missing}}", p)

	// A template that writes to os.Stdout.
	t := template.Must(template.New("out").Parse("done {{.}}\n"))
	if err := t.Execute(os.Stdout, 42); err != nil {
		fmt.Println("stdout:", err)
	}
}
