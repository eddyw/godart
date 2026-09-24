package driver_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTCPKeepAliveUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("socket option assertions require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/keepalive\n\ngo 1.27\n\nrequire example.com/socketopts v0.0.0\nreplace example.com/socketopts => ./socketopts\n")
	netDir := filepath.Join(runtime.GOROOT(), "src", "net")
	for _, item := range []struct {
		name         string
		declarations []string
	}{
		{"tcpconn_keepalive_conf_posix_test.go", []string{"testConfigs"}},
		{"tcpconn_keepalive_posix_test.go", []string{"getCurrentKeepAliveSettings", "verifyKeepAliveSettings"}},
	} {
		source := selectedUpstreamSource(t, filepath.Join(netDir, item.name), item.declarations)
		source = strings.Replace(source, "package net", "package keepalive\nimport . \"net\"", 1)
		source = strings.ReplaceAll(source, `"syscall"`, `"example.com/socketopts"`)
		writeFile(t, filepath.Join(dir, item.name), source)
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	if err := os.Mkdir(filepath.Join(dir, "socketopts"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "socketopts", "go.mod"), "module example.com/socketopts\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "socketopts", "LICENSE"), string(license))
	idle, interval, count, socketLevel, keep := "4", "5", "6", "1", "9"
	if runtime.GOOS == "darwin" {
		idle, interval, count, socketLevel, keep = "16", "257", "258", "65535", "8"
	}
	writeFile(t, filepath.Join(dir, "socketopts", "constants.go"), "package syscall\nconst (SOL_SOCKET="+socketLevel+"; IPPROTO_TCP=6; SO_KEEPALIVE="+keep+"; TCP_KEEPIDLE="+idle+"; TCP_KEEPINTVL="+interval+"; TCP_KEEPCNT="+count+")\n")
	writeFile(t, filepath.Join(dir, "socketopts", "native.go"), `//go:build !dart
package syscall
import "syscall"
func GetsockoptInt(fd, level, option int) (int,error) { return syscall.GetsockoptInt(fd,level,option) }
`)
	writeFile(t, filepath.Join(dir, "socketopts", "dart.go"), `//go:build dart
package syscall
func GetsockoptInt(fd, level, option int) (int,error) { panic("socket option test adapter") }
`)
	writeFile(t, filepath.Join(dir, "native_test.go"), `//go:build !dart
package keepalive
import ("net"; "testing")
func descriptor(t *testing.T, c *net.TCPConn) int {
 raw, err := c.SyscallConn(); if err != nil {t.Fatal(err)}
 var fd int
 if err := raw.Control(func(value uintptr){fd=int(value)}); err!=nil {t.Fatal(err)}
 return fd
}
`)
	writeFile(t, filepath.Join(dir, "dart_test.go"), `//go:build dart
package keepalive
import ("net"; "testing")
func descriptor(t *testing.T, c *net.TCPConn) int { panic("socket descriptor test adapter") }
`)
	writeFile(t, filepath.Join(dir, "keepalive_test.go"), `package keepalive
import ("context"; "errors"; . "net"; "testing"; "time"; stdsyscall "syscall"; "example.com/socketopts")
type fdType = int
const (
 defaultTCPKeepAliveIdle = 15*time.Second
 defaultTCPKeepAliveInterval = 15*time.Second
 defaultTCPKeepAliveCount = 9
 syscall_TCP_KEEPIDLE = syscall.TCP_KEEPIDLE
 syscall_TCP_KEEPINTVL = syscall.TCP_KEEPINTVL
 syscall_TCP_KEEPCNT = syscall.TCP_KEEPCNT
)
func pair(t *testing.T, lc ListenConfig, d Dialer) (*TCPConn,*TCPConn) {
 l,err:=lc.Listen(context.Background(),"tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 defer l.Close()
 client,err:=d.Dial("tcp",l.Addr().String());if err!=nil{t.Fatal(err)}
 t.Cleanup(func(){client.Close()})
 server,err:=l.Accept();if err!=nil{t.Fatal(err)}
 t.Cleanup(func(){server.Close()})
 return client.(*TCPConn),server.(*TCPConn)
}
func TestConfigurations(t *testing.T) {
 baselineClient,baselineServer:=pair(t,ListenConfig{KeepAlive:-1},Dialer{KeepAlive:-1})
 clientDefaults,err:=getCurrentKeepAliveSettings(descriptor(t,baselineClient));if err!=nil{t.Fatal(err)}
 serverDefaults,err:=getCurrentKeepAliveSettings(descriptor(t,baselineServer));if err!=nil{t.Fatal(err)}
 for _,mode:=range []string{"setter","dialer","listener"} {
  t.Run(mode,func(t *testing.T){
   for _,cfg:=range testConfigs {
    lc,d:=ListenConfig{KeepAlive:-1},Dialer{KeepAlive:-1}
    if mode=="dialer"{d.KeepAlive=30*time.Second;d.KeepAliveConfig=cfg}
    if mode=="listener"{lc.KeepAlive=30*time.Second;lc.KeepAliveConfig=cfg}
    client,server:=pair(t,lc,d)
    target,old:=client,clientDefaults
    if mode=="listener"{target,old=server,serverDefaults}
    if mode=="setter"{if err:=target.SetKeepAliveConfig(cfg);err!=nil{t.Fatal(err)}}
    verifyKeepAliveSettings(t,descriptor(t,target),old,cfg)
   }
  })
 }
}
func TestLegacyAndSetters(t *testing.T) {
 for _,idle:=range []time.Duration{-1,0,1,1500*time.Millisecond,30*time.Second} {
  client,server:=pair(t,ListenConfig{KeepAlive:idle},Dialer{KeepAlive:idle})
  for _,c:=range []*TCPConn{client,server} {
   got,err:=getCurrentKeepAliveSettings(descriptor(t,c));if err!=nil{t.Fatal(err)}
   if got.Enable!=(idle>=0){t.Fatalf("idle %v: enabled %v",idle,got.Enable)}
   if idle<0{continue}
   want:=idle;if want==0{want=15*time.Second};want=((want+time.Second-1)/time.Second)*time.Second
   if got.Idle!=want||got.Interval!=15*time.Second||got.Count!=9{t.Fatalf("idle %v: %+v",idle,got)}
  }
 }
 client,_:=pair(t,ListenConfig{},Dialer{})
 for _,idle:=range []time.Duration{1500*time.Millisecond,-1,0,1} {
  old,err:=getCurrentKeepAliveSettings(descriptor(t,client));if err!=nil{t.Fatal(err)}
  if err:=client.SetKeepAlivePeriod(idle);err!=nil{t.Fatal(err)}
  want:=idle;if want<0{want=old.Idle};if want==0{want=15*time.Second};want=((want+time.Second-1)/time.Second)*time.Second
  verifyKeepAliveSettings(t,descriptor(t,client),old,KeepAliveConfig{Enable:old.Enable,Idle:want,Interval:-1,Count:-1})
 }
 cfg:=KeepAliveConfig{Enable:false,Idle:1500*time.Millisecond,Interval:1,Count:3}
 if err:=client.SetKeepAliveConfig(cfg);err!=nil{t.Fatal(err)}
 cfg.Idle=2*time.Second;cfg.Interval=time.Second
 verifyKeepAliveSettings(t,descriptor(t,client),KeepAliveConfig{},cfg)
 if err:=client.SetKeepAlive(true);err!=nil{t.Fatal(err)}
 cfg.Enable=true
 verifyKeepAliveSettings(t,descriptor(t,client),KeepAliveConfig{},cfg)
}
func TestInvalidAndClosed(t *testing.T) {
 var missing *TCPConn
 for _,call:=range []func(){func(){missing.SetKeepAlive(true)},func(){missing.SetKeepAlivePeriod(0)},func(){missing.SetKeepAliveConfig(KeepAliveConfig{})}} {
  func(){defer func(){if recover()==nil{t.Fatal("nil receiver did not panic")}}();call()}()
 }

 for _,c:=range []*TCPConn{new(TCPConn)} {
  for _,err:=range []error{c.SetKeepAlive(true),c.SetKeepAlivePeriod(-1),c.SetKeepAlivePeriod(0),c.SetKeepAliveConfig(KeepAliveConfig{})} {
   if !errors.Is(err,stdsyscall.EINVAL){t.Fatalf("invalid receiver: %v",err)}
  }
 }
 client,_:=pair(t,ListenConfig{},Dialer{})
 client.Close()
 if err:=client.SetKeepAlivePeriod(-1);err!=nil{t.Fatal(err)}
 for _,err:=range []error{client.SetKeepAlive(true),client.SetKeepAlivePeriod(0),client.SetKeepAliveConfig(KeepAliveConfig{})} {
  if !errors.Is(err,ErrClosed){t.Fatalf("closed receiver: %v",err)}
 }
}

`)
	run(t, dir, "go", "test", "-count=1", "-v", ".")
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Tests: true, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	run(t, out, "dart", "pub", "get", "--offline")
	replace := func(path, pattern, replacement string) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rx := regexp.MustCompile(pattern)
		if len(rx.FindAll(data, -1)) != 1 {
			t.Fatalf("adapter does not match %s", path)
		}
		writeFile(t, path, rx.ReplaceAllStringFunc(string(data), func(string) string { return replacement }))
	}
	netPath := filepath.Join(out, "lib", "std", "net.dart")
	data, err := os.ReadFile(netPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, netPath, string(data)+"\nGoInt $testSocketDescriptor(TCPConn$ c) => c.$field$c.$field$handle;\n")
	runtimePath := filepath.Join(out, "goruntime", "lib", "src", "net.dart")
	data, err = os.ReadFile(runtimePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, runtimePath, string(data)+`
int $testSocketOption(int fd,int level,int option) {
 final socket=(_at(fd) as _RawConn).socket;
 final value=RawSocketOption.fromInt(level,option,0);
 final bytes=socket.getRawOption(value);
 return ByteData.sublistView(bytes).getInt32(0,Endian.host);
}
`)
	var descriptorPath, optionsPath string
	if err := filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "dart_test.part.dart" {
			descriptorPath = path
		}
		if filepath.Base(path) == "dart.part.dart" && strings.Contains(path, "socketopts") {
			optionsPath = path
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	replace(descriptorPath, `(?s)GoInt _descriptor\([^\n]*\) \{.*?\n\}`, "GoInt _descriptor($testing.T$? t, $net.TCPConn$? c) { return $net.$testSocketDescriptor(c!); }")
	replace(optionsPath, `(?s)\(GoInt, Iface\?\) GetsockoptInt\([^\n]*\) \{.*?\n\}`, "(GoInt, Iface?) GetsockoptInt(GoInt fd, GoInt level, GoInt option) { return ($gi($testSocketOption($ix(fd),$ix(level),$ix(option))), null); }")
	run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v")
	executable := filepath.Join(out, "keepalive")
	run(t, out, "dart", "compile", "exe", stats.Binaries[0].Entry, "-o", executable)
	run(t, dir, executable, "-test.v")
}
