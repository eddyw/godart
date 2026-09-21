package state

var Steps []string

func Note(name string) { Steps = append(Steps, name) }
