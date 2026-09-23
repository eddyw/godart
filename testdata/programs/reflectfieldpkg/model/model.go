package model

type Record = struct{ x int }

func Read(r Record) int { return r.x }
