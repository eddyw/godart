package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/eddyw/godart/internal/verify"
	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles |
		packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes |
		packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps}
	pkgs, err := packages.Load(cfg, os.Args[1:]...)
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			fmt.Printf("%-24s LOAD ERROR %v\n", p.PkgPath, p.Errors[0])
			continue
		}
		err := verify.Package(p.Fset, p.Syntax, p.TypesInfo)
		if err == nil {
			fmt.Printf("%-24s CLEAN\n", p.PkgPath)
			continue
		}
		causes := map[string]int{}
		total := 0
		for _, line := range strings.Split(err.Error(), "\n") {
			if line == "" {
				continue
			}
			total++
			msg := line
			if k := strings.Index(line, ": "); k >= 0 {
				msg = line[k+2:]
			}
			if k := strings.Index(msg, " ("); k >= 0 {
				msg = msg[:k]
			}
			causes[shorten(msg)]++
		}
		keys := make([]string, 0, len(causes))
		for k := range causes {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return causes[keys[i]] > causes[keys[j]] })
		fmt.Printf("%-24s %d rejections\n", p.PkgPath, total)
		for i, k := range keys {
			if i == 4 {
				break
			}
			fmt.Printf("    %4d  %s\n", causes[k], k)
		}
	}
}

func shorten(s string) string {
	if len(s) > 78 {
		return s[:78]
	}
	return s
}
