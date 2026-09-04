// Program imageuse reads the geometry, the pixel buffers, and the color
// models of image and of image/color. It also reads image/color/palette and
// the draw operations of image/draw.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
)

func main() {
	// The geometry of a point and of a rectangle.
	p := image.Pt(3, 4)
	q := image.Point{X: -1, Y: 2}
	fmt.Println("point:", p, q, p.Add(q), p.Sub(q), p.Mul(2), p.Div(2))
	fmt.Println("point in:", p.In(image.Rect(0, 0, 5, 5)), p.Eq(image.Pt(3, 4)))

	r := image.Rect(1, 2, 7, 9)
	s := image.Rect(4, 5, 12, 12)
	fmt.Println("rect:", r, r.Dx(), r.Dy(), r.Size(), r.Empty())
	fmt.Println("rect ops:", r.Add(p), r.Sub(p), r.Inset(1), r.Intersect(s), r.Union(s))
	fmt.Println("rect test:", r.Overlaps(s), r.In(image.Rect(0, 0, 20, 20)), r.Canon())
	fmt.Println("zero:", image.ZP, image.ZR, image.Rectangle{}.Empty())

	// The color models.
	c := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	cr, cg, cb, ca := c.RGBA()
	fmt.Println("rgba:", cr, cg, cb, ca)
	fmt.Println("gray:", color.GrayModel.Convert(c), color.Gray16Model.Convert(c))
	fmt.Println("nrgba:", color.NRGBAModel.Convert(c), color.RGBA64Model.Convert(c))
	fmt.Println("alpha:", color.AlphaModel.Convert(c), color.Alpha16Model.Convert(c))
	fmt.Println("cmyk:", color.CMYKModel.Convert(c))
	yc, cbv, crv := color.RGBToYCbCr(200, 100, 50)
	fmt.Println("ycbcr:", yc, cbv, crv)
	rr, gg, bb := color.YCbCrToRGB(yc, cbv, crv)
	fmt.Println("back:", rr, gg, bb)
	cc, mm, yy, kk := color.RGBToCMYK(200, 100, 50)
	fmt.Println("cmyk parts:", cc, mm, yy, kk)
	fmt.Println("black and white:", color.Black, color.White, color.Transparent, color.Opaque)

	// The palettes.
	fmt.Println("palette size:", len(palette.Plan9), len(palette.WebSafe))
	pal := color.Palette{color.Black, color.White, c}
	fmt.Println("palette index:", pal.Index(color.RGBA{190, 90, 40, 255}))
	fmt.Println("palette convert:", pal.Convert(color.RGBA{10, 10, 10, 255}))

	// The pixel buffers.
	m := image.NewRGBA(image.Rect(0, 0, 4, 3))
	fmt.Println("rgba bounds:", m.Bounds(), m.Stride, len(m.Pix), m.ColorModel() == color.RGBAModel)
	m.Set(1, 1, c)
	m.SetRGBA(2, 2, color.RGBA{1, 2, 3, 4})
	fmt.Println("at:", m.At(1, 1), m.RGBAAt(2, 2), m.At(9, 9))
	fmt.Println("offset:", m.PixOffset(2, 2), m.Opaque())
	sub := m.SubImage(image.Rect(1, 1, 3, 3)).(*image.RGBA)
	fmt.Println("sub:", sub.Bounds(), sub.At(1, 1))

	g := image.NewGray(image.Rect(0, 0, 3, 3))
	g.SetGray(0, 0, color.Gray{Y: 128})
	fmt.Println("gray image:", g.Bounds(), g.GrayAt(0, 0), g.At(0, 0), g.Opaque())

	n := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	n.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
	fmt.Println("nrgba image:", n.NRGBAAt(0, 0), n.At(0, 0))

	pl := image.NewPaletted(image.Rect(0, 0, 3, 3), pal)
	pl.SetColorIndex(0, 0, 2)
	fmt.Println("paletted:", pl.ColorIndexAt(0, 0), pl.At(0, 0), len(pl.Palette))

	yb := image.NewYCbCr(image.Rect(0, 0, 4, 4), image.YCbCrSubsampleRatio420)
	fmt.Println("ycbcr image:", yb.Bounds(), yb.SubsampleRatio, len(yb.Y), len(yb.Cb))
	fmt.Println("ycbcr offsets:", yb.YOffset(1, 1), yb.COffset(1, 1), yb.At(1, 1))

	cm := image.NewCMYK(image.Rect(0, 0, 2, 2))
	cm.SetCMYK(0, 0, color.CMYK{C: 1, M: 2, Y: 3, K: 4})
	fmt.Println("cmyk image:", cm.CMYKAt(0, 0), cm.Opaque())

	a := image.NewAlpha(image.Rect(0, 0, 2, 2))
	a.SetAlpha(0, 0, color.Alpha{A: 200})
	fmt.Println("alpha image:", a.AlphaAt(0, 0), a.Opaque())

	u := image.NewUniform(color.RGBA{9, 8, 7, 255})
	fmt.Println("uniform:", u.At(100, 100), u.Bounds().Dx() > 0, u.Opaque())

	// The draw operations.
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{20, 40, 60, 255}),
		image.Point{}, draw.Src)
	fmt.Println("after draw:", dst.RGBAAt(0, 0), dst.RGBAAt(3, 3))

	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{200, 0, 0, 128})
	src.SetRGBA(1, 1, color.RGBA{0, 200, 0, 255})
	draw.Draw(dst, image.Rect(1, 1, 3, 3), src, image.Point{}, draw.Over)
	fmt.Println("after over:", dst.RGBAAt(1, 1), dst.RGBAAt(2, 2))

	mask := image.NewAlpha(image.Rect(0, 0, 2, 2))
	mask.SetAlpha(0, 0, color.Alpha{A: 128})
	mask.SetAlpha(1, 1, color.Alpha{A: 255})
	draw.DrawMask(dst, image.Rect(0, 0, 2, 2), image.NewUniform(color.White),
		image.Point{}, mask, image.Point{}, draw.Over)
	fmt.Println("after mask:", dst.RGBAAt(0, 0), dst.RGBAAt(1, 1))

	fmt.Println("ops:", draw.Src, draw.Over)

	var quant draw.Quantizer
	fmt.Println("quantizer nil:", quant == nil)
	var drawer draw.Drawer = draw.FloydSteinberg
	target := image.NewPaletted(image.Rect(0, 0, 4, 4), palette.WebSafe)
	drawer.Draw(target, target.Bounds(), dst, image.Point{})
	fmt.Println("dither:", target.ColorIndexAt(0, 0), target.ColorIndexAt(3, 3))

	// The registry of the decoders.
	_, _, err := image.Decode(nopReader{})
	fmt.Println("decode:", err)
	_, _, err = image.DecodeConfig(nopReader{})
	fmt.Println("decode config:", err)
	fmt.Println("errors:", image.ErrFormat)

	cfg := image.Config{Width: 2, Height: 3, ColorModel: color.RGBAModel}
	fmt.Println("config:", cfg.Width, cfg.Height, cfg.ColorModel == color.RGBAModel)

	fmt.Println("ratio:", image.YCbCrSubsampleRatio444, image.YCbCrSubsampleRatio422,
		image.YCbCrSubsampleRatio420)
}

type nopReader struct{}

func (nopReader) Read(p []byte) (int, error) { return 0, errEOF{} }

type errEOF struct{}

func (errEOF) Error() string { return "EOF" }
