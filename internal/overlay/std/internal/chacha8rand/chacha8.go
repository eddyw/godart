//go:build dart

// This file patches the file of the same name of the package
// internal/chacha8rand.
//
// The real file declares block with no body, and an assembly file of each
// architecture holds that body. It also reads internal/cpu through the package
// unsafe, for 2 offsets that the assembly needs. This file gives block the
// body of block_generic and drops the 2 offsets.
//
// Each other line comes from the real file.
package chacha8rand

import (
	"internal/byteorder"
)

const (
	ctrInc = 4  // increment counter by 4 between block calls
	ctrMax = 16 // reseed when counter reaches 16
	chunk  = 32 // each chunk produced by block is 32 uint64s
	reseed = 4  // reseed with 4 words
)

// block is the chacha8rand block function.
func block(seed *[4]uint64, blocks *[32]uint64, counter uint32) {
	block_generic(seed, blocks, counter)
}

// A State holds the state for a single random generator.
// It must be used from one goroutine at a time.
type State struct {
	buf  [32]uint64
	seed [4]uint64
	i    uint32
	n    uint32
	c    uint32
}

// Next returns the next random value, along with a boolean
// indicating whether one was available.
// If one is not available, the caller should call Refill
// and then repeat the call to Next.
func (s *State) Next() (uint64, bool) {
	i := s.i
	if i >= s.n {
		return 0, false
	}
	s.i = i + 1
	return s.buf[i&31], true // i&31 eliminates bounds check
}

// Init seeds the State with the given seed value.
func (s *State) Init(seed [32]byte) {
	s.Init64([4]uint64{
		byteorder.LEUint64(seed[0*8:]),
		byteorder.LEUint64(seed[1*8:]),
		byteorder.LEUint64(seed[2*8:]),
		byteorder.LEUint64(seed[3*8:]),
	})
}

// Init64 seeds the state with the given seed value.
func (s *State) Init64(seed [4]uint64) {
	s.seed = seed
	block(&s.seed, &s.buf, 0)
	s.c = 0
	s.i = 0
	s.n = chunk
}

// Refill refills the state with more random values.
// After a call to Refill, an immediate call to Next will succeed
// (unless multiple goroutines are incorrectly sharing a state).
func (s *State) Refill() {
	s.c += ctrInc
	if s.c == ctrMax {
		// Reseed with generated uint64s for forward secrecy.
		s.seed[0] = s.buf[len(s.buf)-reseed+0]
		s.seed[1] = s.buf[len(s.buf)-reseed+1]
		s.seed[2] = s.buf[len(s.buf)-reseed+2]
		s.seed[3] = s.buf[len(s.buf)-reseed+3]
		s.c = 0
	}
	block(&s.seed, &s.buf, s.c)
	s.i = 0
	s.n = uint32(len(s.buf))
	if s.c == ctrMax-ctrInc {
		s.n = uint32(len(s.buf)) - reseed
	}
}

// Reseed reseeds the state with new random values.
// After a call to Reseed, any previously returned random values
// have been erased from the memory of the state and cannot be
// recovered.
func (s *State) Reseed() {
	var seed [4]uint64
	for i := range seed {
		for {
			x, ok := s.Next()
			if ok {
				seed[i] = x
				break
			}
			s.Refill()
		}
	}
	s.Init64(seed)
}

// Marshal marshals the state into a byte slice.
func Marshal(s *State) []byte {
	data := make([]byte, 6*8)
	copy(data, "chacha8:")
	used := (s.c/ctrInc)*chunk + s.i
	byteorder.BEPutUint64(data[1*8:], uint64(used))
	for i, seed := range s.seed {
		byteorder.LEPutUint64(data[(2+i)*8:], seed)
	}
	return data
}

type errUnmarshalChaCha8 struct{}

func (*errUnmarshalChaCha8) Error() string {
	return "invalid ChaCha8 encoding"
}

// Unmarshal unmarshals the state from a byte slice.
func Unmarshal(s *State, data []byte) error {
	if len(data) != 6*8 || string(data[:8]) != "chacha8:" {
		return new(errUnmarshalChaCha8)
	}
	used := byteorder.BEUint64(data[1*8:])
	if used > (ctrMax/ctrInc)*chunk-reseed {
		return new(errUnmarshalChaCha8)
	}
	for i := range s.seed {
		s.seed[i] = byteorder.LEUint64(data[(2+i)*8:])
	}
	s.c = ctrInc * (uint32(used) / chunk)
	block(&s.seed, &s.buf, s.c)
	s.i = uint32(used) % chunk
	s.n = chunk
	if s.c == ctrMax-ctrInc {
		s.n = chunk - reseed
	}
	return nil
}
