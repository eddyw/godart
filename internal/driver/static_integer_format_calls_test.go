package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestStaticIntegerFormatCalls(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/formatdiagnostic\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"runtime")
type custom int
func(v custom) Format(s fmt.State,verb rune){fmt.Fprint(s,"custom")}
var calls int
func operand() int {calls++;_,_,line,_:=runtime.Caller(1);fmt.Println("operand line",line);return -17}
func unsignedOperand() uint64 {calls++;return ^uint64(0)}
func suspendedOperand() int {calls++;ch:=make(chan int);go func(){ch<-23}();return <-ch}
func suspendedUnsignedOperand() uint64 {calls++;ch:=make(chan uint64);go func(){ch<-^uint64(0)}();return <-ch}
func panicOperand() int {panic("operand panic")}
func panicFormat(){defer func(){fmt.Println("recovered",recover())}();fmt.Println(fmt.Sprintf("panic %d",panicOperand()))}
func main(){
 fmt.Println(fmt.Sprintf("row %d",operand()),calls)
 fmt.Printf("%q\n",fmt.Sprintf("\xff%d\x00",-9223372036854775808))
 fmt.Println(fmt.Sprintf("%d",uint64(18446744073709551615)))
 fmt.Println(fmt.Sprintf("value %d",unsignedOperand()),calls)
 fmt.Println(fmt.Sprintf("wait %d",suspendedOperand()),calls)
 fmt.Println(fmt.Sprintf("wait %d",suspendedUnsignedOperand()),calls)
 panicFormat()
 fmt.Println(fmt.Sprintf("%d suffix",int8(-128)))
 fmt.Println(fmt.Sprintf("%d",custom(7)),fmt.Sprintf("%03d",7),fmt.Sprintf("%% %d",7))
 f:=fmt.Sprintf;fmt.Println(f("%d",custom(9)))
 values:=[]any{custom(3)};fmt.Println(fmt.Sprintf("%d",values...))
 rows:=make([]string,100000)
 for i:=range rows{rows[i]=fmt.Sprintf("row %d",i)}
 fmt.Println(rows[0],rows[99999],len(rows))
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatal(want)
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
				t.Fatalf("got:\n%s\nwant:\n%s", got.Text, want.Text)
			}
		})
	}
}
