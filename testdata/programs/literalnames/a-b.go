package main

func first() func() int { return func() int { return 1 } }
