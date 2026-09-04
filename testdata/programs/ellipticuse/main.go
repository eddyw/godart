// Command ellipticuse reads the 4 NIST curves of crypto/elliptic. Each answer
// below comes from the parameters of FIPS 186-4 and from a fixed scalar.
package main

import (
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"math/big"
)

func report(name string, c elliptic.Curve) {
	p := c.Params()
	fmt.Println(name, p.Name, p.BitSize)
	fmt.Println(p.P.Text(16))
	fmt.Println(p.N.Text(16))
	fmt.Println(p.B.Text(16))
	fmt.Println(p.Gx.Text(16))
	fmt.Println(p.Gy.Text(16))
	fmt.Println(c.IsOnCurve(p.Gx, p.Gy))

	// The double of the base point, and the base point times 2, agree.
	dx, dy := c.Double(p.Gx, p.Gy)
	sx, sy := c.ScalarBaseMult([]byte{2})
	fmt.Println(dx.Cmp(sx) == 0, dy.Cmp(sy) == 0)
	fmt.Println(dx.Text(16))
	fmt.Println(dy.Text(16))

	// A fixed scalar of 32 bytes, and the point that it names.
	k := make([]byte, len(p.N.Bytes()))
	for i := range k {
		k[i] = byte(i + 1)
	}
	qx, qy := c.ScalarBaseMult(k)
	fmt.Println(qx.Text(16))
	fmt.Println(qy.Text(16))
	fmt.Println(c.IsOnCurve(qx, qy))

	// The uncompressed form and the read of it again.
	enc := elliptic.Marshal(c, qx, qy)
	fmt.Println(hex.EncodeToString(enc))
	rx, ry := elliptic.Unmarshal(c, enc)
	fmt.Println(rx.Cmp(qx) == 0, ry.Cmp(qy) == 0)

	// The compressed form and the read of it again.
	comp := elliptic.MarshalCompressed(c, qx, qy)
	fmt.Println(hex.EncodeToString(comp))
	cx, cy := elliptic.UnmarshalCompressed(c, comp)
	fmt.Println(cx.Cmp(qx) == 0, cy.Cmp(qy) == 0)

	// The sum of 2 points, with the base point and the point of the scalar.
	ax, ay := c.Add(p.Gx, p.Gy, qx, qy)
	fmt.Println(ax.Text(16))
	fmt.Println(ay.Text(16))

	// A point that is not on the curve.
	fmt.Println(c.IsOnCurve(big.NewInt(1), big.NewInt(1)))
	fmt.Println()
}

func main() {
	report("P224", elliptic.P224())
	report("P256", elliptic.P256())
	report("P384", elliptic.P384())
	report("P521", elliptic.P521())
}
