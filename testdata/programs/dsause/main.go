// Command dsause reads the signature of crypto/dsa. The parameters below hold
// 1024 bits and 160 bits, and they stay the same at each run.
//
// The known answer comes from the equations of FIPS 186-4 themselves. The
// program builds 1 signature with math/big and a fixed value of k, and
// crypto/dsa must accept it. It also checks each signature of crypto/dsa
// against the verification equations.
package main

import (
	"crypto/dsa"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"math/big"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func num(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("bad number")
	}
	return n
}

const (
	pHex = "e9e26fb38e617c7a71f9b56f4cb295e2132dc6acfa5e809605b041889b7a0e2f" +
		"57135b0cae15acf992c534425b248cd57712906e9daee1a3369119a2064edcd8" +
		"12077dc84c9d7b45d294271ecf310b46ae2fe76b7b38efe95ea914506fc0a3bc" +
		"f7830231c5a0bce3b205f92fcb0aec3ca9229eebc38eeaaf8fc5d45f74493f5d"
	qHex = "8bc56fa86fce938996cf80fe3bde741067d1e40d"
	gHex = "9c74f2ef05b78390e66e883145e2ff47836038228f56b6e15add4607f199172f" +
		"5d17c6927b51f8b5e55a884db608da47852836eabe5b45671e05e6408e0a1548" +
		"08037caed513924dfb0a340357bd2ec357ef9b97b0877167a1009fca4f852ebf" +
		"0295892514f5a186d52ceeb284c255edae7a08f04dcb7b9c0302b052e8b7e77"
	xHex = "412f63b52cc2f76a14176e2d064e5c393bb03430"
	yHex = "c3574e44bb7c50bcdf45289e604fa60a8c2be0acdeff067e9d04e6ca06e24dcf" +
		"0e2e1e005971235baed125a8e564e3003290b49e00aa97672d2aa813dc662dc2" +
		"ae2991586339c35da6dab9a3a6dd9d26803111b14129164a01ea780e263bef9b" +
		"2c6a210b1e2e2cec87a4259b4891712a2879271c3a283f5be6846efbd8aa9594"
	// kHex names the value of k of the fixed signature below.
	kHex = "2a1f9c3b7d4e5061728394a5b6c7d8e9f0a1b234"
)

// hashInt gives the number of a digest, in the form of FIPS 186-4.
func hashInt(q *big.Int, hash []byte) *big.Int {
	n := (q.BitLen() + 7) / 8
	if len(hash) > n {
		hash = hash[:n]
	}
	return new(big.Int).SetBytes(hash)
}

// signWithK builds the signature of FIPS 186-4 with math/big alone.
func signWithK(p, q, g, x, k, h *big.Int) (*big.Int, *big.Int) {
	r := new(big.Int).Exp(g, k, p)
	r.Mod(r, q)
	kinv := new(big.Int).ModInverse(k, q)
	s := new(big.Int).Mul(x, r)
	s.Add(s, h)
	s.Mul(s, kinv)
	s.Mod(s, q)
	return r, s
}

// verifyEq gives the answer of the verification equations of FIPS 186-4.
func verifyEq(p, q, g, y, r, s, h *big.Int) bool {
	if r.Sign() <= 0 || r.Cmp(q) >= 0 || s.Sign() <= 0 || s.Cmp(q) >= 0 {
		return false
	}
	w := new(big.Int).ModInverse(s, q)
	u1 := new(big.Int).Mul(h, w)
	u1.Mod(u1, q)
	u2 := new(big.Int).Mul(r, w)
	u2.Mod(u2, q)
	v1 := new(big.Int).Exp(g, u1, p)
	v2 := new(big.Int).Exp(y, u2, p)
	v := v1.Mul(v1, v2)
	v.Mod(v, p)
	v.Mod(v, q)
	return v.Cmp(r) == 0
}

func main() {
	p, q, g := num(pHex), num(qHex), num(gHex)
	x, y, k := num(xHex), num(yHex), num(kHex)
	params := dsa.Parameters{P: p, Q: q, G: g}
	pub := dsa.PublicKey{Parameters: params, Y: y}
	priv := dsa.PrivateKey{PublicKey: pub, X: x}
	fmt.Println(p.BitLen(), q.BitLen())

	// The public key comes from the private key.
	fmt.Println(new(big.Int).Exp(g, x, p).Cmp(y) == 0)

	sum := sha1.Sum([]byte("sample"))
	other := sha1.Sum([]byte("test"))
	h := hashInt(q, sum[:])
	ho := hashInt(q, other[:])

	// The fixed signature of the equations, and the answer of crypto/dsa.
	r, s := signWithK(p, q, g, x, k, h)
	fmt.Println(r.Text(16))
	fmt.Println(s.Text(16))
	fmt.Println(dsa.Verify(&pub, sum[:], r, s))
	fmt.Println(dsa.Verify(&pub, other[:], r, s))

	// A signature of crypto/dsa passes the equations, and the check of the
	// package accepts it.
	nr, ns, err := dsa.Sign(rand.Reader, &priv, sum[:])
	must(err)
	fmt.Println(verifyEq(p, q, g, y, nr, ns, h))
	fmt.Println(verifyEq(p, q, g, y, nr, ns, ho))
	fmt.Println(dsa.Verify(&pub, sum[:], nr, ns))
	fmt.Println(dsa.Verify(&pub, other[:], nr, ns))

	// A new key of the same parameters signs and verifies.
	nk := dsa.PrivateKey{PublicKey: dsa.PublicKey{Parameters: params}}
	must(dsa.GenerateKey(&nk, rand.Reader))
	fmt.Println(nk.X.Cmp(q) < 0, nk.Y.Cmp(p) < 0)
	fmt.Println(new(big.Int).Exp(g, nk.X, p).Cmp(nk.Y) == 0)
	kr, ks, err := dsa.Sign(rand.Reader, &nk, sum[:])
	must(err)
	fmt.Println(dsa.Verify(&nk.PublicKey, sum[:], kr, ks))

	// A signature that holds 0 fails the check.
	fmt.Println(dsa.Verify(&pub, sum[:], big.NewInt(0), s))
	fmt.Println(dsa.Verify(&pub, sum[:], r, big.NewInt(0)))
	fmt.Println(dsa.Verify(&pub, sum[:], q, s))

	// The 4 sizes of the parameters hold the numbers of FIPS 186-3.
	fmt.Println(dsa.L1024N160, dsa.L2048N224, dsa.L2048N256, dsa.L3072N256)
	fmt.Println(dsa.ErrInvalidPublicKey)
}
