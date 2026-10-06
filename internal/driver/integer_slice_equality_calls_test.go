package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestIntegerSliceEqualityCalls(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/integerequality\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"math";"runtime";"slices")
type Word uint64
type Words []Word
var order int
func operand(tag int, value Words) Words {
 order=order*10+tag
 _,_,line,_:=runtime.Caller(1)
 fmt.Println("operand",tag,line)
 return value
}
func pair() (Words, Words) {
 order=order*10+3
 return Words{^Word(0),1<<63}, Words{^Word(0),1<<63}
}
func generic[T ~int|~uint64](a,b []T) bool { return slices.Equal(a,b) }
func main() {
 a:=Words{^Word(0),1<<63,1}
 b:=append(Words(nil),a...)
 fmt.Println("direct",slices.Equal(operand(1,a),operand(2,b)),order)
 fmt.Println("spread",slices.Equal(pair()),order)
 fmt.Println("empty",slices.Equal(Words(nil),Words{}))
 fmt.Println("length",slices.Equal(a,b[:2]))
 b[1]--
 fmt.Println("different",slices.Equal(a,b))
 fmt.Println("shared",slices.Equal(a[1:],a[1:]))
 equal:=slices.Equal[Words,Word]
 fmt.Println("function",equal(a,a),equal(a,b))
 fmt.Println("generic",generic([]int{-1,2},[]int{-1,2}),generic([]Word{^Word(0)},[]Word{^Word(0)}))
 fmt.Println("byte",slices.Equal([]byte{0,255},[]byte{0,255}),slices.Equal([]int8{-128,127},[]int8{-128,127}))
 fmt.Println("zero size",slices.Equal(make([]struct{},1<<60),make([]struct{},(1<<60)-1)))
 fmt.Println("float",slices.Equal([]float64{math.NaN()},[]float64{math.NaN()}))
 func(){defer func(){fmt.Println("interface panic",recover()!=nil)}();slices.Equal([]any{[]int{1}},[]any{[]int{1}})}()
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("equality calls exit=%d\ngot:\n%s\nwant:\n%s", got.Exit, got.Text, want.Text)
			}
		})
	}
}
