//go:build dart

package main

//gart:dart package:goruntime/goruntime.dart $drop
func drop(fn func() bool)
