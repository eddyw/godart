package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	dir, err := os.Getwd()
	fmt.Println(err == nil, len(dir) > 0)

	tmp := os.TempDir() + "/gart_osuse"
	os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp+"/deep", 0o755); err != nil {
		fmt.Println("mkdir", err)
		return
	}

	path := tmp + "/a.txt"
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		fmt.Println("write", err)
		return
	}
	data, err := os.ReadFile(path)
	fmt.Println(string(data) == "hello\n", err == nil)

	info, err := os.Stat(path)
	fmt.Println(info.Name(), info.Size(), info.IsDir(), err == nil)

	dinfo, err := os.Stat(tmp + "/deep")
	fmt.Println(dinfo.IsDir(), err == nil)

	_, err = os.ReadFile(tmp + "/missing.txt")
	fmt.Println(err != nil, os.IsNotExist(err))

	_, err = os.Stat(tmp + "/missing.txt")
	fmt.Println(os.IsNotExist(err))

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("open", err)
		return
	}
	buf := make([]byte, 3)
	n, err := f.Read(buf)
	fmt.Println(n, string(buf[:n]), err == nil)
	n, err = f.Read(buf)
	fmt.Println(n, string(buf[:n]), err == nil)
	n, err = f.Read(buf)
	fmt.Println(n, err != nil)
	f.Close()

	// A read after a close fails.
	_, err = f.Read(buf)
	fmt.Println(err != nil)

	w, err := os.Create(tmp + "/b.txt")
	fmt.Println(err == nil)
	w.WriteString("one\n")
	w.Write([]byte("two\n"))
	w.Close()
	b, _ := os.ReadFile(tmp + "/b.txt")
	fmt.Println(string(b) == "one\ntwo\n")

	// A write at the end keeps the content.
	a, err := os.OpenFile(tmp+"/b.txt", syscall.O_WRONLY|syscall.O_APPEND, 0o644)
	fmt.Println(err == nil)
	a.WriteString("three\n")
	a.Close()
	b, _ = os.ReadFile(tmp + "/b.txt")
	fmt.Println(string(b) == "one\ntwo\nthree\n")

	// A seek moves the read position.
	r, _ := os.Open(tmp + "/b.txt")
	pos, err := r.Seek(4, 0) // 0 is io.SeekStart
	fmt.Println(pos, err == nil)
	got := make([]byte, 4)
	n, _ = r.Read(got)
	fmt.Println(string(got[:n]))
	r.Close()

	if err := os.Rename(tmp+"/b.txt", tmp+"/c.txt"); err != nil {
		fmt.Println("rename", err)
	}
	_, err = os.Stat(tmp + "/c.txt")
	fmt.Println(err == nil)

	fmt.Println(os.Remove(tmp+"/c.txt") == nil)

	// gart run merges standard error into standard output, thus the test writes
	// only to standard output.
	fmt.Fprintf(os.Stdout, "to stdout %d\n", 7)
	fmt.Fprintln(os.Stdout, "a line")

	fmt.Println(os.Getenv("GART_OSUSE_MISSING") == "")
	_, found := os.LookupEnv("GART_OSUSE_MISSING")
	fmt.Println(found)
	fmt.Println(len(os.Environ()) > 0)
	fmt.Println(len(os.Args) > 0)

	os.RemoveAll(tmp)
	_, err = os.Stat(tmp)
	fmt.Println(os.IsNotExist(err))
	fmt.Println(os.RemoveAll(tmp) == nil)
}
