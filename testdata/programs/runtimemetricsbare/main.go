package main

import (
	"fmt"
	"os"
	"runtime/metrics"
)

const name = "/godebug/non-default-behavior/panicnil:events"

func count() uint64 {
	samples := []metrics.Sample{{Name: name}}
	metrics.Read(samples)
	return samples[0].Value.Uint64()
}

func panicNil() {
	defer func() { recover() }()
	panic(nil)
}

func wrongKind(read func()) {
	defer func() { fmt.Println(recover()) }()
	read()
}

func main() {
	found := false
	for _, description := range metrics.All() {
		if description.Name == name {
			found = description.Kind == metrics.KindUint64 && description.Cumulative && description.Description != ""
		}
	}
	if !found {
		panic("missing counter description")
	}
	start := count()
	os.Setenv("GODEBUG", "panicnil=1")
	panicNil()
	panicNil()
	if count()-start != 0 {
		panic("missing non-default events")
	}
	os.Setenv("GODEBUG", "panicnil=0")
	panicNil()
	if count()-start != 0 {
		panic("default panic changed counter")
	}
	samples := []metrics.Sample{{Name: name}}
	metrics.Read(samples)
	samples[0].Name = "/unknown:events"
	metrics.Read(samples)
	if samples[0].Value.Kind() != metrics.KindBad {
		panic("unknown metric retained a value")
	}
	var value metrics.Value
	wrongKind(func() { value.Uint64() })
	wrongKind(func() { value.Float64() })
	wrongKind(func() { value.Float64Histogram() })
	metrics.Read(nil)
	fmt.Println("counter events and value kinds match")
}
