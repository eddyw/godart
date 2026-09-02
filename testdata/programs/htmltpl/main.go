package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"
)

type page struct {
	Title   string
	Comment string
	Link    string
	Items   []string
}

const src = `<h1>{{.Title}}</h1>
<p>{{.Comment}}</p>
<a href="{{.Link}}">go</a>
<script>var name = "{{.Title}}";</script>
<ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul>
`

func main() {
	t, err := template.New("page").Parse(src)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	p := page{
		Title:   `A & B <tag>`,
		Comment: `a "quoted" word and a <script> word`,
		Link:    "https://example.com/?a=1&b=2",
		Items:   []string{"first", "<second>", "third & last"},
	}
	if err := t.Execute(os.Stdout, p); err != nil {
		fmt.Println("execute:", err)
		return
	}

	safe := template.Must(template.New("safe").Parse(`{{.}}`))
	var sb strings.Builder
	if err := safe.Execute(&sb, template.HTML("<b>raw</b>")); err != nil {
		fmt.Println("execute safe:", err)
		return
	}
	fmt.Println("raw:", sb.String())

	bad := template.Must(template.New("bad").Parse(`<a href="{{.}}">x</a>`))
	var sb2 strings.Builder
	if err := bad.Execute(&sb2, "javascript:alert(1)"); err != nil {
		fmt.Println("execute bad:", err)
		return
	}
	fmt.Println("blocked:", sb2.String())
}
