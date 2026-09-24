package main

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

func entry(name string, inode uint64) []byte {
	offset := int(unsafe.Offsetof(syscall.Dirent{}.Name))
	size := (offset + len(name) + 1 + 7) &^ 7
	buf := make([]byte, size)
	binary.LittleEndian.PutUint64(buf, inode)
	binary.LittleEndian.PutUint16(buf[unsafe.Offsetof(syscall.Dirent{}.Reclen):], uint16(size))
	if len(syscall.Dirent{}.Name) == 1024 {
		binary.LittleEndian.PutUint16(buf[18:], uint16(len(name)))
	}
	copy(buf[offset:], name)
	return buf
}

func main() {
	var data []byte
	for i, name := range []string{".", "..", "ascii", "世界", "\xff", "gone", "last"} {
		inode := uint64(i + 1)
		if name == "gone" {
			inode = 0
		}
		data = append(data, entry(name, inode)...)
	}
	for _, max := range []int{-1, 0, 1, 2, 20} {
		consumed, count, names := syscall.ParseDirent(data, max, []string{"prefix"})
		fmt.Printf("max %d %d %d %x\n", max, consumed, count, names)
	}
	for size := 0; size <= len(data); size++ {
		consumed, count, names := syscall.ParseDirent(data[:size], -1, nil)
		fmt.Printf("short %d %d %d %x\n", size, consumed, count, names)
	}
	for _, size := range []uint16{0, 1, 8, 18, 19, 20, 21, 65535} {
		func() {
			defer func() { fmt.Println("panic", recover()) }()
			buf := entry("value", 1)
			binary.LittleEndian.PutUint16(buf[unsafe.Offsetof(syscall.Dirent{}.Reclen):], size)
			consumed, count, names := syscall.ParseDirent(buf, -1, nil)
			fmt.Printf("record %d %d %d %x\n", size, consumed, count, names)
		}()
	}
}
