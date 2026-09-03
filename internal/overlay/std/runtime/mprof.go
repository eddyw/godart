//go:build dart

package runtime

import (
	"strings"
	"syscall/dart"
)

// The profiles of the runtime.
//
// The scheduler of gart lives in the Dart runtime library, and it holds the
// stack of each goroutine. Thus the profile of the goroutines carries real
// frames. The Dart VM counts no allocation, no block, and no contention of a
// mutex, thus the 3 other profiles stay out. Read docs/support.md.

// A StackRecord holds the stack of 1 record of a profile.
type StackRecord struct {
	// Stack0 holds the program counter of each frame. The list ends at the
	// first entry that is 0.
	Stack0 [32]uintptr
}

// Stack gives the frames of the record, a prefix of Stack0.
func (r *StackRecord) Stack() []uintptr {
	for i, pc := range r.Stack0 {
		if pc == 0 {
			return r.Stack0[0:i]
		}
	}
	return r.Stack0[0:]
}

// GoroutineProfile writes 1 record for each live goroutine into p, and it gives
// the count of the goroutines.
//
// A list that is shorter than the count takes no write, and the flag is then
// false. The first record is the goroutine that calls.
//
// A goroutine that gives up control leaves the Dart stack, thus its record
// holds the frames of the state that resumes. Read docs/parity.md.
func GoroutineProfile(p []StackRecord) (n int, ok bool) {
	parked := dart.Parked()
	n = len(parked) + 1
	if n > len(p) {
		return n, false
	}
	// Frame 0 is this function, thus the skip starts at 1.
	fillRecord(&p[0], dart.Stack(1))
	for i, block := range parked {
		// The head of a block holds the identifier and the reason, and each
		// line after it holds 1 frame.
		_, rest, _ := strings.Cut(block, "\n")
		var frames []string
		for rest != "" {
			var frame string
			frame, rest, _ = strings.Cut(rest, "\n")
			frames = append(frames, frame)
		}
		fillRecord(&p[i+1], frames)
	}
	return n, true
}

// fillRecord writes the program counter of each frame into a record, and it
// clears the rest of the list.
func fillRecord(r *StackRecord, frames []string) {
	n := 0
	for _, frame := range frames {
		if n == len(r.Stack0) {
			break
		}
		r.Stack0[n] = record(frame)
		n++
	}
	for i := n; i < len(r.Stack0); i++ {
		r.Stack0[i] = 0
	}
}

// CPUProfile gives the next part of the profile of the processor.
//
// Go removed the operation, and each call of it panics. gart holds the same
// text, thus a program reads the same report.
func CPUProfile() []byte {
	panic("CPUProfile no longer available")
}
