package main

import (
	"fmt"
	mrand "math/rand"
	rand "math/rand/v2"
)

// The seed of each generator is fixed, thus the values of go run and of the
// Dart build must be equal.

func pcg() {
	r := rand.New(rand.NewPCG(1, 2))
	fmt.Println("pcg uint64:", r.Uint64(), r.Uint64(), r.Uint64())
	fmt.Println("pcg int:", r.IntN(100), r.IntN(100), r.IntN(1))
	fmt.Println("pcg int32:", r.Int32N(1000), r.Int32(), r.Uint32N(7))
	fmt.Println("pcg int64:", r.Int64N(1<<40), r.Int64(), r.Uint64N(11))
	fmt.Println("pcg float:", r.Float64(), r.Float32())
	fmt.Println("pcg norm:", r.NormFloat64(), r.ExpFloat64())
	fmt.Println("pcg perm:", r.Perm(8))

	list := []int{0, 1, 2, 3, 4, 5, 6, 7}
	r.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
	fmt.Println("pcg shuffle:", list)

	// N takes a type parameter over each integer type.
	fmt.Println("pcg n:", r.N(int8(100)), r.N(uint16(1000)), r.N(uint64(1<<40)))
}

func chacha() {
	var seed [32]byte
	for i := range seed {
		seed[i] = byte(i)
	}
	r := rand.New(rand.NewChaCha8(seed))
	fmt.Println("chacha uint64:", r.Uint64(), r.Uint64(), r.Uint64())
	fmt.Println("chacha int:", r.IntN(1000), r.IntN(1000))
	fmt.Println("chacha float:", r.Float64())

	// The state holds 32 words, thus 40 reads pass the refill.
	total := uint64(0)
	for i := 0; i < 40; i++ {
		total += r.Uint64()
	}
	fmt.Println("chacha total:", total)
}

func marshal() {
	var seed [32]byte
	for i := range seed {
		seed[i] = byte(255 - i)
	}
	c := rand.NewChaCha8(seed)
	c.Uint64()
	data, err := c.MarshalBinary()
	fmt.Println("marshal:", err == nil, len(data))

	var back rand.ChaCha8
	if err := back.UnmarshalBinary(data); err != nil {
		fmt.Println("unmarshal:", err)
		return
	}
	fmt.Println("marshal same:", c.Uint64() == back.Uint64())

	p := rand.NewPCG(7, 9)
	p.Uint64()
	pdata, err := p.MarshalBinary()
	fmt.Println("pcg marshal:", err == nil, len(pdata))
	var pback rand.PCG
	if err := pback.UnmarshalBinary(pdata); err != nil {
		fmt.Println("pcg unmarshal:", err)
		return
	}
	fmt.Println("pcg marshal same:", p.Uint64() == pback.Uint64())
}

func zipf() {
	r := rand.New(rand.NewPCG(3, 4))
	z := rand.NewZipf(r, 2.0, 1.0, 100)
	fmt.Println("zipf:", z.Uint64(), z.Uint64(), z.Uint64())
}

func global() {
	// The top-level functions read a source that the clock seeds, thus the
	// program prints only the range of a value.
	fmt.Println("global n range:", rand.N(int32(50)) < 50, rand.N(uint(7)) < 7)
	n := rand.IntN(10)
	fmt.Println("global range:", n >= 0 && n < 10)
	f := rand.Float64()
	fmt.Println("global float range:", f >= 0 && f < 1)
}

// version1 reads the first form of the package. A seeded generator holds the
// lagged Fibonacci algorithm of rng.go, thus the values of Go and of the Dart
// build must be equal.
func version1() {
	r := mrand.New(mrand.NewSource(1))
	fmt.Println("v1 int:", r.Intn(100), r.Int31n(1000), r.Int63n(1<<40))
	fmt.Println("v1 raw:", r.Int63(), r.Uint32(), r.Uint64())
	fmt.Println("v1 float:", r.Float64(), r.Float32())
	fmt.Println("v1 norm:", r.NormFloat64(), r.ExpFloat64())
	fmt.Println("v1 perm:", r.Perm(8))

	b := make([]byte, 12)
	n, err := r.Read(b)
	fmt.Println("v1 read:", n, err == nil, b)

	list := []int{0, 1, 2, 3, 4, 5, 6, 7}
	r.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
	fmt.Println("v1 shuffle:", list)

	// A seed of the same value gives the same sequence again.
	r.Seed(1)
	fmt.Println("v1 reseed:", r.Intn(100))

	z := mrand.NewZipf(mrand.New(mrand.NewSource(5)), 2.0, 1.0, 100)
	fmt.Println("v1 zipf:", z.Uint64(), z.Uint64())

	// The top-level Seed is a no-op since Go 1.24, thus the program prints
	// only the range of a value.
	mrand.Seed(42)
	fmt.Println("v1 global range:", mrand.Intn(10) < 10)
}

func main() {
	pcg()
	chacha()
	marshal()
	zipf()
	global()
	version1()
}
