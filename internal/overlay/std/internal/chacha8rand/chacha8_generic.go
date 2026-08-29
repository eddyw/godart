//go:build dart

// This file patches the file of the same name of the package
// internal/chacha8rand.
//
// The real file reads a `[16][4]uint32` and a `[32]uint64` as 2 views of 1
// memory block, through the package unsafe. The Dart runtime holds no such
// view, thus setup writes each 32-bit word and block_generic packs 2 words
// into each 64-bit word of the buffer. The values stay equal, because the
// target of the compiler is little endian.
//
// Each other line comes from the real file.
package chacha8rand

// setup sets up 4 ChaCha8 blocks in b32 with the counter and seed.
// Note that b32 is [16][4]uint32 not [4][16]uint32: the blocks are interlaced
// the same way they would be in a 4-way SIMD implementations.
func setup(seed *[4]uint64, b32 *[16][4]uint32, counter uint32) {
	// Constants; same as in ChaCha20: "expand 32-byte k"
	for i := 0; i < 4; i++ {
		b32[0][i] = 0x61707865
		b32[1][i] = 0x3320646e
		b32[2][i] = 0x79622d32
		b32[3][i] = 0x6b206574
	}

	// Seed values. Each 64-bit word of the seed gives 2 rows, and each row
	// holds 1 value in each of the 4 lanes.
	for i := 0; i < 4; i++ {
		low := uint32(seed[i])
		high := uint32(seed[i] >> 32)
		for lane := 0; lane < 4; lane++ {
			b32[4+2*i][lane] = low
			b32[5+2*i][lane] = high
		}
	}

	// Counters.
	for i := 0; i < 4; i++ {
		b32[12][i] = counter + uint32(i)
	}

	// Zeros.
	for i := 0; i < 4; i++ {
		b32[13][i] = 0
		b32[14][i] = 0
		b32[15][i] = 0
	}
}

func _() {
	// block and block_generic must have same type
	x := block
	x = block_generic
	_ = x
}

// block_generic is the non-assembly block implementation,
// for use on systems without special assembly.
func block_generic(seed *[4]uint64, buf *[32]uint64, counter uint32) {
	var b [16][4]uint32

	setup(seed, &b, counter)

	for i := range b[0] {
		// Load block i from b[*][i] into local variables.
		b0 := b[0][i]
		b1 := b[1][i]
		b2 := b[2][i]
		b3 := b[3][i]
		b4 := b[4][i]
		b5 := b[5][i]
		b6 := b[6][i]
		b7 := b[7][i]
		b8 := b[8][i]
		b9 := b[9][i]
		b10 := b[10][i]
		b11 := b[11][i]
		b12 := b[12][i]
		b13 := b[13][i]
		b14 := b[14][i]
		b15 := b[15][i]

		// 4 iterations of eight quarter-rounds each is 8 rounds
		for round := 0; round < 4; round++ {
			b0, b4, b8, b12 = qr(b0, b4, b8, b12)
			b1, b5, b9, b13 = qr(b1, b5, b9, b13)
			b2, b6, b10, b14 = qr(b2, b6, b10, b14)
			b3, b7, b11, b15 = qr(b3, b7, b11, b15)

			b0, b5, b10, b15 = qr(b0, b5, b10, b15)
			b1, b6, b11, b12 = qr(b1, b6, b11, b12)
			b2, b7, b8, b13 = qr(b2, b7, b8, b13)
			b3, b4, b9, b14 = qr(b3, b4, b9, b14)
		}

		// Store block i back into b[*][i].
		// Add b4..b11 back to the original key material,
		// like in ChaCha20, to avoid trivial invertibility.
		// There is no entropy in b0..b3 and b12..b15
		// so we can skip the additions and save some time.
		b[0][i] = b0
		b[1][i] = b1
		b[2][i] = b2
		b[3][i] = b3
		b[4][i] += b4
		b[5][i] += b5
		b[6][i] += b6
		b[7][i] += b7
		b[8][i] += b8
		b[9][i] += b9
		b[10][i] += b10
		b[11][i] += b11
		b[12][i] = b12
		b[13][i] = b13
		b[14][i] = b14
		b[15][i] = b15
	}

	// Pack each row into 2 words of the buffer. The real file writes the
	// same bytes through a view of the memory.
	for i := 0; i < 16; i++ {
		buf[2*i] = uint64(b[i][0]) | uint64(b[i][1])<<32
		buf[2*i+1] = uint64(b[i][2]) | uint64(b[i][3])<<32
	}
}

// qr is the (inlinable) ChaCha8 quarter round.
func qr(a, b, c, d uint32) (_a, _b, _c, _d uint32) {
	a += b
	d ^= a
	d = d<<16 | d>>16
	c += d
	b ^= c
	b = b<<12 | b>>20
	a += b
	d ^= a
	d = d<<8 | d>>24
	c += d
	b ^= c
	b = b<<7 | b>>25
	return a, b, c, d
}
