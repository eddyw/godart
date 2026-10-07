package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestOpaquePointerCanonicalIdentity(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/opaquepointer\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"reflect";"unsafe")
type T struct { X int; Y int }
type Z struct { Array [0]int; N int }
type S struct { First int; Pad byte; Sub T; Array [3]T; Last int }
type V struct { First int `+"`tag:\"other\"`"+`; Pad byte; Sub T; Array [3]T; Last int }
type P unsafe.Pointer
func allocate[T any](v T)*T{return &v}
func opaque[T any](p *T)unsafe.Pointer{return unsafe.Pointer(p)}
func main(){
 s:=new(S)
 fmt.Println("first field",unsafe.Pointer(s)==unsafe.Pointer(&s.First))
 fmt.Println("last field",unsafe.Pointer(s)!=unsafe.Pointer(&s.Last))
 fmt.Println("nested field",unsafe.Pointer(&s.Sub)==unsafe.Pointer(&s.Sub.X),unsafe.Pointer(&s.Sub)!=unsafe.Pointer(&s.Sub.Y))
 first:=unsafe.Pointer(&s.Array[0].X);m:=map[unsafe.Pointer]int{first:17}
 fmt.Println("array aliases",unsafe.Pointer(&s.Array)==first,unsafe.Pointer(&s.Array[0])==first,m[unsafe.Pointer(&s.Array)])
 s.Array[0]=T{X:29};fmt.Println("assignment hash",m[unsafe.Pointer(&s.Array[0].X)],first==unsafe.Pointer(&s.Array[0]))
 v:=(*V)(s);fmt.Println("view aliases",unsafe.Pointer(v)==unsafe.Pointer(s),unsafe.Pointer(&v.Sub.X)==unsafe.Pointer(&s.Sub))
 copy:=*s;fmt.Println("copy allocation",unsafe.Pointer(&copy)!=unsafe.Pointer(s),unsafe.Pointer(&copy.Array[0])!=first)
 sl:=make([]T,3,3);sl[0]=T{X:1};sl[1]=T{X:2}
 ap:=(*[2]T)(sl[1:]);fmt.Println("slice view",unsafe.Pointer(ap)==unsafe.Pointer(&sl[1]),unsafe.Pointer(&ap[0].X)==unsafe.Pointer(&sl[1]))
 slm:=map[unsafe.Pointer]int{unsafe.Pointer(&sl[1].Y):23};sl[1]=T{X:3,Y:4};fmt.Println("slice hash",slm[unsafe.Pointer(&sl[1].Y)])
 old:=unsafe.Pointer(&sl[0]);sl=append(sl,T{X:5});fmt.Println("slice allocation",old!=unsafe.Pointer(&sl[0]))
 gp:=allocate(T{X:7});fmt.Println("generic box",opaque(gp)==unsafe.Pointer(&gp.X),opaque(&gp.Y)==unsafe.Pointer(&gp.Y))
 gm:=map[unsafe.Pointer]int{opaque(gp):31};*gp=T{X:8};fmt.Println("generic assignment",gm[unsafe.Pointer(&gp.X)],gp.X)
 ai:=allocate([2]T{});fmt.Println("generic array",opaque(ai)==unsafe.Pointer(&ai[0].X))
 n:=new(int);fmt.Println("scalar box",unsafe.Pointer(n)==opaque(n))
 fmt.Println("interface keys",any(P(unsafe.Pointer(s)))==any(P(unsafe.Pointer(&s.First))))
 field:=reflect.ValueOf(s).Elem().Field(4).Addr().Interface().(*int)
 fmt.Println("reflection field",unsafe.Pointer(field)==unsafe.Pointer(&s.Last),unsafe.Pointer(field)!=unsafe.Pointer(s))
 z:=new(Z);zm:=map[unsafe.Pointer]int{unsafe.Pointer(&z.Array):19};other:=new(Z);other.N=1
 fmt.Println("empty field identity",unsafe.Pointer(z)==unsafe.Pointer(&z.Array),zm[unsafe.Pointer(&z.Array)],zm[unsafe.Pointer(&other.Array)],other.N)
 fmt.Println("reverse aliases",unsafe.Pointer(&s.First)==unsafe.Pointer(s),unsafe.Pointer(&s.Sub.X)==unsafe.Pointer(&s.Sub),unsafe.Pointer(&ap[0].X)==unsafe.Pointer(ap))
 var za [3]struct{};zap:=unsafe.Pointer(&za);zam:=map[unsafe.Pointer]int{zap:37}
 fmt.Println("zero element alias",zap==unsafe.Pointer(&za[0]),zam[unsafe.Pointer(&za[2])])
 zs:=make([]struct{},3);zsp:=unsafe.Pointer(&zs[1]);zsm:=map[unsafe.Pointer]int{zsp:43}
 fmt.Println("zero slice hash",zsm[unsafe.Pointer(&zs[1])])
 ga0:=allocate([0]int{});fmt.Println("generic empty array",opaque(ga0)==unsafe.Pointer(ga0))
 fmt.Println("formatted aliases",fmt.Sprint(unsafe.Pointer(s))==fmt.Sprint(unsafe.Pointer(&s.First)))
 rs:=reflect.MakeSlice(reflect.TypeOf([]T{}),3,3).Interface().([]T)
 rp:=(*[2]T)(rs[1:]);rk:=map[unsafe.Pointer]int{unsafe.Pointer(&rs[1].X):47}
 fmt.Println("reflect slice allocation",unsafe.Pointer(rp)==unsafe.Pointer(&rs[1]),rk[unsafe.Pointer(rp)],unsafe.Pointer(&rp[0].X)==unsafe.Pointer(rp))
 ra:=reflect.New(reflect.ArrayOf(3,reflect.TypeOf(T{}))).Interface().(*[3]T)
 ram:=map[unsafe.Pointer]int{unsafe.Pointer(&ra[0].X):53}
 fmt.Println("reflect array allocation",unsafe.Pointer(ra)==unsafe.Pointer(&ra[0]),ram[unsafe.Pointer(ra)])
 av:=T{X:61};grown:=append(make([]T,1,1),av);gv:=(*[1]T)(grown[1:]);gk:=map[unsafe.Pointer]int{unsafe.Pointer(&grown[1].X):59}
 fmt.Println("append allocation",unsafe.Pointer(gv)==unsafe.Pointer(&grown[1]),gk[unsafe.Pointer(gv)],unsafe.Pointer(&av)!=unsafe.Pointer(&grown[1]))
 spread:=append(make([]T,1,1),[]T{{X:67},{X:71}}...);sv:=(*[2]T)(spread[1:]);sm:=map[unsafe.Pointer]int{unsafe.Pointer(&spread[1].X):61}
 fmt.Println("append spread allocation",unsafe.Pointer(sv)==unsafe.Pointer(&spread[1]),sm[unsafe.Pointer(sv)])
 var nilptr *int;fmt.Println("nil pointer",opaque(nilptr)==nil)
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
