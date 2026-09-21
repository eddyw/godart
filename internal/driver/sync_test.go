package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// The synchronous guarantee of milestone 2. A named function type that carries
// //gart:sync names a position that a host outside Go calls for a result. A
// function that can block gives the zero value there, and no report reaches the
// program, thus the build must reject it.

// buildSync compiles 1 file and gives the error of the build.
func buildSync(t *testing.T, src string) error {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syncpos\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), src)
	return driver.Build(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
	})
}

func TestSyncFieldRejectsABlockingLiteral(t *testing.T) {
	err := buildSync(t, `package main

//gart:sync
type Builder func() int

type Node struct {
	Build Builder
}

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	n := Node{Build: func() int { return <-ch }}
	println(n.Build())
}
`)
	if err == nil {
		t.Fatal("the build accepted a function that blocks in a synchronous position")
	}
	for _, want := range []string{
		"this function can block",
		"main.Builder is a synchronous position",
		"Build at",
		"a channel receive at",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the report holds no %q:\n%v", want, err)
		}
	}
}

func TestSyncFieldAcceptsAPlainLiteral(t *testing.T) {
	err := buildSync(t, `package main

//gart:sync
type Builder func() int

type Node struct {
	Build Builder
}

func main() {
	n := Node{Build: func() int { return 1 }}
	println(n.Build())
}
`)
	if err != nil {
		t.Fatalf("the build rejected a function that never blocks: %v", err)
	}
}

// A callback with no result still accepts a function that blocks. The host
// wraps such a value as a goroutine, thus the value never returns to Dart.
func TestSyncLeavesAPlainCallbackAlone(t *testing.T) {
	err := buildSync(t, `package main

//gart:sync
type Builder func() int

type Node struct {
	Build   Builder
	OnPress func()
}

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	n := Node{
		Build:   func() int { return 2 },
		OnPress: func() { println(<-ch) },
	}
	println(n.Build())
	n.OnPress()
}
`)
	if err != nil {
		t.Fatalf("the build rejected a callback with no result: %v", err)
	}
}

func TestSyncParameterRejectsABlockingLiteral(t *testing.T) {
	err := buildSync(t, `package main

//gart:sync
type Builder func() int

func run(b Builder) int { return b() }

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	println(run(func() int { return <-ch }))
}
`)
	if err == nil {
		t.Fatal("the build accepted a function that blocks in a synchronous parameter")
	}
	if want := "b at"; !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// The cause of the block stands 1 call away from the function value. The hint
// must name the operation of the body, and not the call.
func TestSyncNamesTheIndirectCause(t *testing.T) {
	err := buildSync(t, `package main

//gart:sync
type Builder func() int

func read(ch chan int) int { return <-ch }

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	var b Builder = func() int { return read(ch) }
	println(b())
}
`)
	if err == nil {
		t.Fatal("the build accepted an indirect function that blocks")
	}
	if want := "a channel receive at"; !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// A named function that blocks reaches the position through a method value.
func TestSyncRejectsABlockingMethod(t *testing.T) {
	err := buildSync(t, `package main

//gart:sync
type Builder func() int

type Source struct{ ch chan int }

func (s *Source) Read() int { return <-s.ch }

func main() {
	s := &Source{ch: make(chan int)}
	go func() { s.ch <- 1 }()
	var b Builder = s.Read
	println(b())
}
`)
	if err == nil {
		t.Fatal("the build accepted a method that blocks in a synchronous position")
	}
	if want := "main.Builder is a synchronous position"; !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// The directive stands in a dependency module. The load must give the syntax of
// that module, because the check reads the doc comment of the declaration.
func TestSyncReadsADirectiveOfADependency(t *testing.T) {
	ui, err := filepath.Abs("../../flutter/ui")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syncdep\n\ngo 1.27.0\n\n"+
		"require gart.dev/ui v0.0.0\n\nreplace gart.dev/ui => "+ui+"\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "gart.dev/ui"

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	n := ui.Watch{Build: func() ui.Node {
		return ui.Text{Data: "row"}
	}}
	bad := ui.Watch{Build: func() ui.Node {
		<-ch
		return ui.Text{Data: "row"}
	}}
	println(n.Build() != nil, bad.Build() != nil)
}
`)
	err = driver.Build(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
	})
	if err == nil {
		t.Fatal("the build accepted a function that blocks in ui.Builder")
	}
	if want := "ui.Builder is a synchronous position"; !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
	if n := strings.Count(err.Error(), "ui.Builder is a synchronous position"); n != 1 {
		t.Errorf("the report names %d functions, and only 1 blocks:\n%v", n, err)
	}
}

func TestSyncRejectsTheDirectiveOnAnotherType(t *testing.T) {
	err := buildSync(t, `package main

//gart:sync
type Count int

func main() {
	println(int(Count(1)))
}
`)
	if err == nil {
		t.Fatal("the build accepted the directive on a type that is not a function type")
	}
	if want := "is not a function type"; !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// A type declaration inside a group carries its own doc comment.
func TestSyncReadsADirectiveInAGroup(t *testing.T) {
	err := buildSync(t, `package main

type (
	// Plain takes no marker.
	Plain func() int

	//gart:sync
	Builder func() int
)

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	var p Plain = func() int { return <-ch }
	var b Builder = func() int { return <-ch }
	println(p(), b())
}
`)
	if err == nil {
		t.Fatal("the build accepted a function that blocks in a synchronous position")
	}
	if want := "main.Builder"; !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
	if want := "main.Plain"; strings.Contains(err.Error(), want) {
		t.Errorf("the report names a type that carries no directive:\n%v", err)
	}
}

// TestSyncAcceptsAnErrorFormatWithHTTP holds the shape that broke the 3 Flutter
// examples on 2026-09-03. The flow resolves an interface method call to each
// method of that name and that signature, thus a program that holds 1
// fmt.Formatter lets fmt.State.Write reach io.PipeWriter.Write, which selects.
// A build method that only formats an error then reached a rejection.
//
// crypto/rand held Int and Prime for a few hours of that day, and the 2 read
// math/big. Each program that reads crypto/tls thus carried a Formatter, and
// each Flutter example that reads net/http stopped building. Read the section
// "The method group of an interface call" of docs/parity.md.
func TestSyncAcceptsAnErrorFormatWithHTTP(t *testing.T) {
	err := buildSync(t, `package main

import (
	"errors"
	"fmt"
	"net/http"
)

//gart:sync
type Builder func() string

type Node struct {
	Build Builder
}

var failure = errors.New("the call failed")

func text() string {
	if err := failure; err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("%v", http.StatusOK)
}

func main() {
	n := Node{Build: text}
	println(n.Build())
}
`)
	if err != nil {
		t.Fatalf("the build rejected a function that only formats an error:\n%v", err)
	}
}

func TestSyncAcceptsABigFormatWithHTTP(t *testing.T) {
	err := buildSync(t, `package main

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"
)

//gart:sync
type Builder func() string

type Node struct {
	Build Builder
}

var failure = errors.New("the call failed")

func text() string {
	n := new(big.Int).Exp(big.NewInt(2), big.NewInt(64), nil)
	if err := failure; err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("%v %v", n, http.StatusOK)
}

func main() {
	n := Node{Build: text}
	println(n.Build())
}
`)
	if err != nil {
		t.Fatalf("the build rejected a function that only formats a big.Int and an error:\n%v", err)
	}
}

// The narrow group of a type assertion must keep each target that the
// assertion can give. A type that implements the asserted interface and the
// interface of the value stays in the group, thus a block inside it still
// reaches the check.
func TestSyncRejectsABlockingTargetOfATypeAssertion(t *testing.T) {
	err := buildSync(t, `package main

import "io"

//gart:sync
type Builder func() string

type Node struct {
	Build Builder
}

type slow struct{ ch chan int }

func (s *slow) Write(p []byte) (int, error) { return len(p), nil }

func (s *slow) WriteString(text string) (int, error) {
	<-s.ch
	return len(text), nil
}

func emit(w io.Writer) string {
	if sw, ok := w.(io.StringWriter); ok {
		sw.WriteString("x")
	}
	return "done"
}

func text() string { return emit(&slow{ch: make(chan int)}) }

func main() {
	n := Node{Build: text}
	println(n.Build())
}
`)
	if err == nil {
		t.Fatal("the build accepted a function that reaches a Write that blocks")
	}
}

func TestSyncFormatRejectsBlockingOperands(t *testing.T) {
	for name, operand := range map[string]string{
		"value":     "slow{}",
		"field":     "struct{ V slow }{slow{}}",
		"slice":     "[]slow{{}}",
		"map":       "map[int]slow{1: {}}",
		"interface": "any(slow{})",
	} {
		t.Run(name, func(t *testing.T) {
			err := buildSync(t, `package main
import "fmt"
//gart:sync
type Builder func() string
type slow struct{}
func (slow) String() string { <-make(chan int); return "" }
func text() string { return fmt.Sprintf("%v", `+operand+`) }
var build Builder = text
func main() { println(build()) }
`)
			if err == nil {
				t.Fatal("accepted a formatter that can block")
			}
		})
	}
}

func TestSyncInterfaceOriginRejectsMutableValues(t *testing.T) {
	for name, setup := range map[string]string{
		"assignment":   "var value error = safe(); func init() { value = slow{} }",
		"address":      "var value error = safe(); func init() { change(&value) }; func change(p *error) { *p = slow{} }",
		"named-result": "var value error = factory(); func factory() (e error) { defer func() { e = slow{} }(); return quiet{} }",
		"parameter":    "var value error = identity(slow{}); func identity(e error) error { return e }",
	} {
		t.Run(name, func(t *testing.T) {
			err := buildSync(t, `package main
//gart:sync
type Builder func() string
type quiet struct{}
func (quiet) Error() string { return "quiet" }
func safe() error { return quiet{} }
type slow struct{}
func (slow) Error() string { <-make(chan int); return "" }
`+setup+`
func text() string { return value.Error() }
var build Builder = text
func main() { println(build()) }
`)
			if err == nil {
				t.Fatal("accepted an interface method that can block")
			}
		})
	}
}

func TestSyncGenericZeroRejectsBlockingMethods(t *testing.T) {
	err := buildSync(t, `package main
//gart:sync
type Builder func() string
type slow int
func (slow) Error() string { <-make(chan int); return "" }
func text[T error]() string { var value T; return value.Error() }
var build Builder = text[slow]
func main() { println(build()) }
`)
	if err == nil || !strings.Contains(err.Error(), "this function can block") {
		t.Fatalf("expected a synchronous callback rejection, got %v", err)
	}
}

func TestSyncFormatRejectsBlockingPanicValue(t *testing.T) {
	err := buildSync(t, `package main
import "fmt"
//gart:sync
type Builder func() string
type slow struct{}
func (slow) String() string { <-make(chan int); return "" }
type broken struct{}
func (broken) String() string { panic(slow{}) }
func text() string { return fmt.Sprintf("%v", broken{}) }
var build Builder = text
func main() { println(build()) }
`)
	if err == nil {
		t.Fatal("accepted a formatter whose panic value can block")
	}
}

func TestSyncAcceptsInterfaceErrorText(t *testing.T) {
	err := buildSync(t, `package main
import (
 "fmt"
 "net/http"
)
//gart:sync
type Builder func() string
var failure error
func text() string { return fmt.Sprintf("error: %q %% %s", failure, http.StatusText(200)) }
var build Builder = text
func main() { println(build()) }
`)
	if err != nil {
		t.Fatalf("rejected synchronous interface error text: %v", err)
	}
}

func TestSyncInterfaceFormatRejectsBlockingMethods(t *testing.T) {
	for name, methods := range map[string]string{
		"error":     `func (value) Error() string { <-make(chan int); return "" }`,
		"formatter": `func (value) Error() string { return "" }; func (value) Format(fmt.State, rune) { <-make(chan int) }`,
		"panic":     `func (value) Error() string { panic(slow{}) }; type slow struct{}; func (slow) String() string { <-make(chan int); return "" }`,
	} {
		t.Run(name, func(t *testing.T) {
			err := buildSync(t, `package main
import "fmt"
//gart:sync
type Builder func() string
type value struct{}
`+methods+`
var failure error = value{}
func text() string { return fmt.Sprintf("error: %q", failure) }
var build Builder = text
func main() { println(build()) }
`)
			if err == nil {
				t.Fatal("accepted an error formatter that can block")
			}
		})
	}
}

func TestSyncFormatRejectsMutableOrigins(t *testing.T) {
	for name, setup := range map[string]string{
		"assignment":   "var value error; func init() { value = slow{} }",
		"address":      "var value error; func init() { change(&value) }; func change(p *error) { *p = slow{} }",
		"named-result": "var value error = factory(); func factory() (e error) { defer func() { e = slow{} }(); return nil }",
		"parameter":    "var value error = identity(slow{}); func identity(e error) error { return e }",
	} {
		t.Run(name, func(t *testing.T) {
			err := buildSync(t, `package main
import "fmt"
//gart:sync
type Builder func() string
type slow struct{}
func (slow) Error() string { <-make(chan int); return "" }
`+setup+`
func text() string { return fmt.Sprintf("%s", value) }
var build Builder = text
func main() { println(build()) }
`)
			if err == nil {
				t.Fatal("accepted a mutable format operand that can block")
			}
		})
	}
}

func TestSyncOnceCallbacks(t *testing.T) {
	for _, test := range []struct {
		name, declaration, body string
		rejected                bool
	}{
		{"function", "var cached = sync.OnceFunc(func() {})", "cached(); return 1", false},
		{"value", "var cached = sync.OnceValue(func() int { return 7 })", "return cached()", false},
		{"values", "var cached = sync.OnceValues(func() (int, bool) { return 7, true })", "value, _ := cached(); return value", false},
		{"independent", "var cached = sync.OnceValue(func() int { return 7 }); var slow = sync.OnceValue(func() int { <-make(chan int); return 0 })", "return cached()", false},
		{"blocking function", "var cached = sync.OnceFunc(func() { <-make(chan int) })", "cached(); return 1", true},
		{"blocking value", "var cached = sync.OnceValue(func() int { <-make(chan int); return 7 })", "return cached()", true},
		{"blocking values", "var cached = sync.OnceValues(func() (int, bool) { <-make(chan int); return 7, true })", "value, _ := cached(); return value", true},
		{"mutable result", "var cached = sync.OnceValue(func() int { return 7 }); func init() { cached = sync.OnceValue(func() int { <-make(chan int); return 0 }) }", "return cached()", true},
		{"mutable callback", "var callback = func() int { return 7 }; var cached = sync.OnceValue(func() int { return callback() }); func init() { callback = func() int { <-make(chan int); return 0 } }", "return cached()", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := buildSync(t, "package main\nimport \"sync\"\n//gart:sync\ntype Builder func() int\n"+test.declaration+"\nvar build Builder = func() int { "+test.body+" }\nfunc main() { println(build()) }\n")
			if test.rejected {
				if err == nil || !strings.Contains(err.Error(), "this function can block") {
					t.Fatalf("expected callback rejection, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
