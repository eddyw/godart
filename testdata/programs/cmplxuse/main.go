package main

import (
	"fmt"
	"math"
	"math/cmplx"
)

func main() {
	z := complex(3.0, 4.0)
	fmt.Println(cmplx.Abs(z))
	fmt.Println(cmplx.Conj(z))
	fmt.Printf("%.6f\n", cmplx.Phase(z))

	r, theta := cmplx.Polar(z)
	fmt.Printf("%.6f %.6f\n", r, theta)
	fmt.Printf("%.6f\n", cmplx.Rect(r, theta))

	fmt.Printf("%.6f\n", cmplx.Sqrt(complex(-1.0, 0.0)))
	fmt.Printf("%.6f\n", cmplx.Exp(complex(0.0, math.Pi)))
	fmt.Printf("%.6f\n", cmplx.Log(complex(math.E, 0.0)))
	fmt.Printf("%.6f\n", cmplx.Pow(complex(2.0, 0.0), complex(10.0, 0.0)))

	fmt.Printf("%.6f %.6f %.6f\n", cmplx.Sin(z), cmplx.Cos(z), cmplx.Tan(z))
	fmt.Printf("%.6f %.6f\n", cmplx.Sinh(z), cmplx.Cosh(z))
	fmt.Printf("%.6f %.6f\n", cmplx.Asin(z), cmplx.Acos(z))
	fmt.Printf("%.6f\n", cmplx.Atan(z))

	fmt.Println(cmplx.IsNaN(cmplx.NaN()), cmplx.IsInf(cmplx.Inf()))
	fmt.Println(cmplx.IsNaN(z), cmplx.IsInf(z))
	fmt.Printf("%.6f\n", cmplx.Cot(z))
}
