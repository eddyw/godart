package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// The program leaves ioutil.ReadDir out. That function opens the directory as
// a file, and Dart gives no handle of a directory. docs/parity.md holds the
// difference in the row of the os package.
func main() {
	dir, err := ioutil.TempDir("", "ioutiluse")
	if err != nil {
		fmt.Println("tempdir:", err)
		return
	}
	defer os.RemoveAll(dir)

	name := filepath.Join(dir, "one.txt")
	if err := ioutil.WriteFile(name, []byte("hello file"), 0o644); err != nil {
		fmt.Println("write:", err)
		return
	}
	data, err := ioutil.ReadFile(name)
	fmt.Println("read:", string(data), err == nil)

	all, err := ioutil.ReadAll(strings.NewReader("all of it"))
	fmt.Println("readall:", string(all), err == nil)

	f, err := ioutil.TempFile(dir, "two")
	fmt.Println("tempfile:", err == nil, strings.HasPrefix(filepath.Base(f.Name()), "two"))
	f.Close()

	n, err := ioutil.Discard.Write([]byte("gone"))
	fmt.Println("discard:", n, err == nil)

	rc := ioutil.NopCloser(strings.NewReader("wrapped"))
	got, _ := ioutil.ReadAll(rc)
	fmt.Println("nopcloser:", string(got), rc.Close() == nil)

	missing, err := ioutil.ReadFile(filepath.Join(dir, "absent.txt"))
	fmt.Println("missing:", len(missing), os.IsNotExist(err))
}
