package main

func other() func() int { return func() int { return 2 } }
