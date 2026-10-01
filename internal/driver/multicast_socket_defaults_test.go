package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMulticastSocketDefaults(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native multicast options require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/multicastdefaults\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"net";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 interfaces,err:=net.Interfaces();must(err)
 var loopback *net.Interface
 for _,iface:=range interfaces{if iface.Flags&net.FlagLoopback!=0{loopback=&iface;break}}
 if loopback==nil{panic("no loopback interface")}
 for _,iface:=range []*net.Interface{nil,loopback}{
  conn,err:=net.ListenMulticastUDP("udp4",iface,&net.UDPAddr{IP:net.IPv4(239,255,42,1)});must(err)
  raw,err:=conn.SyscallConn();must(err)
  must(raw.Control(func(fd uintptr){
   loop,err:=syscall.GetsockoptInt(int(fd),syscall.IPPROTO_IP,syscall.IP_MULTICAST_LOOP);must(err)
   address,err:=syscall.GetsockoptInet4Addr(int(fd),syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF);must(err)
   fmt.Println("loopback",loop);fmt.Printf("interface %d.%d.%d.%d\n",address[0],address[1],address[2],address[3])
  }))
  must(conn.Close())
 }
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	out := t.TempDir()
	if err := os.CopyFS(filepath.Join(out, "lib"), os.DirFS(filepath.Join(runtimeDir(t), "lib"))); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pubspec.yaml", "LICENSE"} {
		data, err := os.ReadFile(filepath.Join(runtimeDir(t), name))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(out, name), string(data))
	}
	entry := filepath.Join("lib", "src", "net.dart")
	source, err := os.ReadFile(filepath.Join(out, entry))
	if err != nil {
		t.Fatal(err)
	}
	// A runtime copy exposes actual socket options without a production inspection API.
	writeFile(t, filepath.Join(out, entry), string(source)+`
Future<void> main(List<String> args) async {
  final interfaces = await NetworkInterface.list(includeLoopback: true, type: InternetAddressType.IPv4);
  final loopback = interfaces.firstWhere((n) => n.addresses.any((a) => a.isLoopback));
  for (final name in ['', loopback.name]) {
    final packet = await _bindShared('0.0.0.0', 0);
    try {
      await _joinGroup(packet, '239.255.42.1', name);
      print('loopback ${packet.socket.multicastLoopback ? 1 : 0}');
      final option = RawSocketOption(RawSocketOption.levelIPv4, RawSocketOption.IPv4MulticastInterface, Uint8List(4));
      print('interface ${packet.socket.getRawOption(option).join('.')}');
    } finally {
      packet.socket.close();
      await packet.closed.future;
    }
  }
}
`)
	run(t, out, "dart", "pub", "get")
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, out, "dart", entry)
			} else {
				got = runUpstreamTarget(t, target, out, entry, out, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nDart runtime (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
