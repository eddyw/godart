// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package metrics

import (
	"internal/godebugs"
	"math"
	"strings"
	"syscall/dart"
)

type ValueKind int

const (
	KindBad ValueKind = iota
	KindUint64
	KindFloat64
	KindFloat64Histogram
)

type Value struct {
	kind      ValueKind
	scalar    uint64
	histogram *Float64Histogram
}

func (v Value) Kind() ValueKind { return v.kind }

func (v Value) Uint64() uint64 {
	if v.kind != KindUint64 {
		panic("called Uint64 on non-uint64 metric value")
	}
	return v.scalar
}

func (v Value) Float64() float64 {
	if v.kind != KindFloat64 {
		panic("called Float64 on non-float64 metric value")
	}
	return math.Float64frombits(v.scalar)
}

func (v Value) Float64Histogram() *Float64Histogram {
	if v.kind != KindFloat64Histogram {
		panic("called Float64Histogram on non-Float64Histogram metric value")
	}
	return v.histogram
}

type Float64Histogram struct {
	Counts  []uint64
	Buckets []float64
}

type Sample struct {
	Name  string
	Value Value
}

type Description struct {
	Name        string
	Description string
	Kind        ValueKind
	Cumulative  bool
}

var allDesc = func() []Description {
	var descriptions []Description
	for _, info := range godebugs.All {
		if !info.Opaque {
			descriptions = append(descriptions, Description{
				Name: "/godebug/non-default-behavior/" + info.Name + ":events",
				Description: "The number of non-default behaviors executed by the " +
					info.Package + " package " + "due to a non-default " +
					"GODEBUG=" + info.Name + "=... setting.",
				Kind:       KindUint64,
				Cumulative: true,
			})
		}
	}
	return descriptions
}()

// All lists the supported GODEBUG event counters. Native Go heap, scheduler,
// and collector measurements have no equivalent in this runtime.
func All() []Description { return allDesc }

func Read(samples []Sample) {
	const prefix = "/godebug/non-default-behavior/"
	const suffix = ":events"
	for i := range samples {
		samples[i].Value = Value{}
		name := samples[i].Name
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		name = strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
		info := godebugs.Lookup(name)
		if info != nil && !info.Opaque {
			samples[i].Value = Value{kind: KindUint64, scalar: dart.GODEBUGCount(name)}
		}
	}
}
