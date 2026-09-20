package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"syscall"
)

func main() {
	syscall.Clearenv()
	fmt.Println(syscall.Environ(), os.Environ())
	_ = syscall.Setenv("SHARED", "first")
	fmt.Println(os.LookupEnv("SHARED"))
	_ = os.Setenv("SHARED", "second")
	fmt.Println(syscall.Getenv("SHARED"))
	_ = syscall.Setenv("EMPTY", "")
	fmt.Println(os.LookupEnv("EMPTY"))
	_ = os.Unsetenv("SHARED")
	_ = syscall.Setenv("SHARED", "third")
	fmt.Println(syscall.Environ(), os.Environ())
	values := syscall.Environ()
	values[0] = "CHANGED=yes"
	fmt.Println(reflect.DeepEqual(values, syscall.Environ()))
	for _, pair := range [][2]string{{"", "x"}, {"A=B", "x"}, {"A\x00", "x"}, {"A", "x\x00"}} {
		err := syscall.Setenv(pair[0], pair[1])
		var wrapped *os.SyscallError
		fmt.Println(err == syscall.EINVAL, errors.As(os.Setenv(pair[0], pair[1]), &wrapped), wrapped.Err == syscall.EINVAL)
	}
	_ = syscall.Unsetenv("EMPTY")
	fmt.Println(os.LookupEnv("EMPTY"))
	os.Clearenv()
	fmt.Println(syscall.Environ(), os.Environ())
	_ = os.Setenv("SHARED", "last")
	fmt.Println(syscall.Environ())
}
