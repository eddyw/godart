package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinKevent(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Kevent requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/kevent\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "errors"
 "fmt"
 "os"
 "reflect"
 "syscall"
)

func event(label string, q int, changes []syscall.Kevent_t, timeout *syscall.Timespec, expected uint64) {
 saved := append([]syscall.Kevent_t(nil), changes...)
 events := []syscall.Kevent_t{{Ident:999,Filter:127,Flags:65535,Fflags:0xffffffff,Data:-99},{Ident:998,Filter:126,Flags:65534,Fflags:0xfffffffe,Data:-98}}
 before := append([]syscall.Kevent_t(nil),events...)
 n,err:=syscall.Kevent(q,changes,events,timeout)
 code:=syscall.Errno(0);if err!=nil {code=err.(syscall.Errno)}
 fmt.Println(label,n,int(code),reflect.DeepEqual(changes,saved),reflect.DeepEqual(events,before))
 for i:=0;i<n;i++ {v:=events[i];fmt.Println("event",i,v.Ident==expected,v.Filter,v.Flags,v.Fflags,v.Data,v.Udata==nil)}
 fmt.Println("tail",events[1]==before[1])
}

func main(){
 if len(os.Args)>1 { unsupported(); return }
 q,err:=syscall.Kqueue();if err!=nil{panic(err)};defer syscall.Close(q)
 zero:=&syscall.Timespec{}
 event("empty",q,nil,zero,0)
 event("bad fd",-1,nil,zero,0)
 event("bad seconds",q,nil,&syscall.Timespec{Sec:-1},0)
 event("bad nanos",q,nil,&syscall.Timespec{Nsec:1000000000},0)
 event("user add",q,[]syscall.Kevent_t{{Ident:1,Filter:syscall.EVFILT_USER,Flags:syscall.EV_ADD|syscall.EV_CLEAR|syscall.EV_RECEIPT,Data:(1<<40)+7}},zero,1)
 event("user trigger",q,[]syscall.Kevent_t{{Ident:1,Filter:syscall.EVFILT_USER,Fflags:syscall.NOTE_TRIGGER|syscall.NOTE_FFCOPY|0x123456}},zero,1)
 event("user drained",q,nil,zero,0)
 event("user delete",q,[]syscall.Kevent_t{{Ident:1,Filter:syscall.EVFILT_USER,Flags:syscall.EV_DELETE|syscall.EV_RECEIPT}},zero,1)
 event("missing receipt",q,[]syscall.Kevent_t{{Ident:1,Filter:syscall.EVFILT_USER,Flags:syscall.EV_DELETE|syscall.EV_RECEIPT}},zero,1)
 event("missing error",q,[]syscall.Kevent_t{{Ident:1,Filter:syscall.EVFILT_USER,Flags:syscall.EV_DELETE}},zero,1)
 event("bad filter",q,[]syscall.Kevent_t{{Ident:2,Filter:127,Flags:syscall.EV_ADD|syscall.EV_RECEIPT}},zero,2)
 r,w,err:=os.Pipe();if err!=nil{panic(err)};defer r.Close();defer w.Close();fd:=int(r.Fd())
 event("pipe add",q,[]syscall.Kevent_t{{Ident:uint64(fd),Filter:syscall.EVFILT_READ,Flags:syscall.EV_ADD|syscall.EV_CLEAR|syscall.EV_RECEIPT}},zero,uint64(fd))
 if _,err:=w.Write([]byte{1,2,3});err!=nil{panic(err)}
 event("pipe bytes",q,nil,zero,uint64(fd))
 data:=make([]byte,3);if n,err:=r.Read(data);err!=nil||n!=3{panic("pipe read")};fmt.Println("bytes",data)
 event("pipe drained",q,nil,zero,uint64(fd))
 if err:=w.Close();err!=nil{panic(err)}
 event("pipe eof",q,nil,zero,uint64(fd))
 boundaries()
 overlaps()
}

func overlaps(){
 for _,direction:=range []int{1,-1}{
  q,err:=syscall.Kqueue();if err!=nil{panic(err)}
  values:=[]syscall.Kevent_t{{Ident:101,Filter:syscall.EVFILT_USER,Flags:syscall.EV_ADD|syscall.EV_RECEIPT,Data:-11},{Ident:102,Filter:syscall.EVFILT_USER,Flags:syscall.EV_DELETE|syscall.EV_RECEIPT,Data:-12},{Ident:103,Filter:syscall.EVFILT_USER,Flags:syscall.EV_ADD|syscall.EV_RECEIPT,Data:-13}}
  changes,events:=values[:2],values[1:];if direction<0{changes,events=values[1:],values[:2]}
  n,err:=syscall.Kevent(q,changes,events,&syscall.Timespec{});code:=syscall.Errno(0);if err!=nil{code=err.(syscall.Errno)}
  fmt.Println("overlap",direction,n,int(code))
  for _,v:=range values{fmt.Println(v.Ident,v.Filter,v.Flags,v.Fflags,v.Data,v.Udata==nil)}
  syscall.Close(q)
 }
}


func boundaries(){
 v:=syscall.Kevent_t{Ident:0xfedcba9876543210,Data:-9223372036854775807}
 q,err:=syscall.Kqueue();if err!=nil{panic(err)};defer syscall.Close(q)
 zero:=&syscall.Timespec{}
 changes:=[]syscall.Kevent_t{{Ident:v.Ident,Filter:syscall.EVFILT_USER,Flags:syscall.EV_ADD|syscall.EV_RECEIPT,Data:v.Data}}
 events:=make([]syscall.Kevent_t,1)
 n,err:=syscall.Kevent(q,changes,events,nil)
 fmt.Println("high ident",n,err==nil,events[0].Ident==v.Ident,events[0].Filter,events[0].Flags,events[0].Data,events[0].Udata==nil,changes[0].Data==v.Data)
 alias:=[]syscall.Kevent_t{{Ident:29,Filter:syscall.EVFILT_USER,Flags:syscall.EV_ADD|syscall.EV_RECEIPT,Data:-99},{Ident:30,Filter:syscall.EVFILT_USER,Flags:syscall.EV_DELETE|syscall.EV_RECEIPT,Data:-98}}
 n,err=syscall.Kevent(q,alias,alias,zero)
 fmt.Println("alias",n,err==nil,alias[0].Ident,alias[0].Filter,alias[0].Flags,alias[0].Data,alias[1].Ident,alias[1].Filter,alias[1].Flags,alias[1].Data)
 changes=[]syscall.Kevent_t{{Ident:31,Filter:syscall.EVFILT_USER,Flags:syscall.EV_ADD}}
 n,err=syscall.Kevent(q,changes,nil,nil)
 fmt.Println("no events",n,err==nil,changes[0].Flags==syscall.EV_ADD)
 n,err=syscall.Kevent(q,[]syscall.Kevent_t{},[]syscall.Kevent_t{},zero)
 fmt.Println("empty slices",n,err==nil)
}


func unsupported(){
 token:=byte(19)
 cases:=[]struct{name string;changes,events []syscall.Kevent_t}{
  {"changes",[]syscall.Kevent_t{{Ident:1,Udata:&token}},[]syscall.Kevent_t{{Ident:2}}},
  {"events",[]syscall.Kevent_t{{Ident:3}},[]syscall.Kevent_t{{Ident:4,Udata:&token}}},
  {"tail",nil,[]syscall.Kevent_t{{Ident:5},{Ident:6,Udata:&token}}},
 }
 for _,c:=range cases {
  changes:=append([]syscall.Kevent_t(nil),c.changes...);events:=append([]syscall.Kevent_t(nil),c.events...)
  n,err:=syscall.Kevent(-1,c.changes,c.events,&syscall.Timespec{})
  code,typed:=err.(syscall.Errno)
  intact:=reflect.DeepEqual(changes,c.changes)&&reflect.DeepEqual(events,c.events)&&token==19
  match:=n==-1&&typed&&code==syscall.ENOSYS&&errors.Is(err,errors.ErrUnsupported)&&intact
  fmt.Println(c.name,n,int(code),match)
  if !match {panic("unsupported Kevent pointer contract")}
 }
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	if want.Exit != 0 || want.Text != "empty 0 0 true true\ntail true\nbad fd -1 9 true true\ntail true\nbad seconds -1 22 true true\ntail true\nbad nanos 0 0 true true\ntail true\nuser add 1 0 true false\nevent 0 true -10 16481 0 0 true\ntail true\nuser trigger 1 0 true false\nevent 0 true -10 97 1193046 0 true\ntail true\nuser drained 0 0 true true\ntail true\nuser delete 1 0 true false\nevent 0 true -10 16450 0 0 true\ntail true\nmissing receipt 1 0 true false\nevent 0 true -10 16450 0 2 true\ntail true\nmissing error 1 0 true false\nevent 0 true -10 16386 0 2 true\ntail true\nbad filter 1 0 true false\nevent 0 true 127 16449 0 22 true\ntail true\npipe add 1 0 true false\nevent 0 true -1 16481 0 0 true\ntail true\npipe bytes 1 0 true false\nevent 0 true -1 97 0 3 true\ntail true\nbytes [1 2 3]\npipe drained 0 0 true true\ntail true\npipe eof 1 0 true false\nevent 0 true -1 32865 0 0 true\ntail true\nhigh ident 1 true true -10 16449 0 true true\nalias 2 true 29 -10 16449 0 30 -10 16450 2\nno events 0 true true\nempty slices 0 true\noverlap 1 2 0\n101 -10 65 0 -11 true\n101 -10 16449 0 0 true\n101 -10 16449 0 0 true\noverlap -1 2 0\n102 -10 16450 0 2 true\n103 -10 16449 0 0 true\n103 -10 65 0 -13 true" {
		t.Fatalf("original native Kevent: %+v", want)
	}
	t.Log("original native Kevent", want.Text)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			modes := []string{"unsupported"}
			if target == "vm" || target == "aot" {
				modes = append([]string{"native"}, modes...)
			}
			for _, mode := range modes {
				t.Run(mode, func(t *testing.T) {
					args := []string{}
					expected := want
					if mode == "unsupported" {
						args = []string{mode}
						expected = programResult{Exit: 0, Text: "changes -1 78 true\nevents -1 78 true\ntail -1 78 true"}
					}
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", append([]string{filepath.Join(out, "bin/main.dart")}, args...)...)
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, args)
					}
					if got != expected {
						t.Fatalf("Kevent %s differs: Go=%+v Gart=%+v", mode, expected, got)
					}
				})
			}
		})
	}
}
