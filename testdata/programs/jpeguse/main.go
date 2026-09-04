// Program jpeguse writes a JPEG image and reads it again. It shows the
// encoder and the decoder. It also shows the quality option and the error
// types of image/jpeg.
package main

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
)

// source builds a small image with a gradient.
func source(w, h int) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetRGBA(x, y, color.RGBA{
				R: uint8(x * 16),
				G: uint8(y * 16),
				B: uint8(255 - (x+y)*8),
				A: 255,
			})
		}
	}
	return m
}

func main() {
	src := source(16, 16)

	for _, q := range []int{1, jpeg.DefaultQuality, 100} {
		var b bytes.Buffer
		if err := jpeg.Encode(&b, src, &jpeg.Options{Quality: q}); err != nil {
			fmt.Println("encode:", err)
			return
		}
		raw := b.Bytes()
		fmt.Printf("quality %3d: %d bytes crc=%08x head=%x\n", q, len(raw),
			crc32.ChecksumIEEE(raw), raw[:4])
	}

	var b bytes.Buffer
	if err := jpeg.Encode(&b, src, nil); err != nil {
		fmt.Println("encode default:", err)
		return
	}
	raw := b.Bytes()
	fmt.Printf("default: %d bytes crc=%08x\n", len(raw), crc32.ChecksumIEEE(raw))

	m, err := jpeg.Decode(bytes.NewReader(raw))
	fmt.Println("decode err:", err, "bounds:", m.Bounds())
	ym := m.(*image.YCbCr)
	fmt.Println("ratio:", ym.SubsampleRatio, "y len:", len(ym.Y), "cb len:", len(ym.Cb))
	for _, p := range []image.Point{{0, 0}, {7, 7}, {15, 15}} {
		r, g, bl, a := m.At(p.X, p.Y).RGBA()
		fmt.Printf("pixel %v: %d %d %d %d\n", p, r, g, bl, a)
	}

	cfg, err := jpeg.DecodeConfig(bytes.NewReader(raw))
	fmt.Println("config:", cfg.Width, cfg.Height, err)

	m2, name, err := image.Decode(bytes.NewReader(raw))
	fmt.Println("registry:", name, err, m2.Bounds())
	cfg2, name2, err := image.DecodeConfig(bytes.NewReader(raw))
	fmt.Println("registry config:", name2, cfg2.Width, cfg2.Height, err)

	// A gray image takes the gray path of the encoder.
	g := image.NewGray(image.Rect(0, 0, 8, 8))
	for i := range g.Pix {
		g.Pix[i] = uint8(i * 3)
	}
	var gb bytes.Buffer
	if err := jpeg.Encode(&gb, g, &jpeg.Options{Quality: 90}); err != nil {
		fmt.Println("encode gray:", err)
		return
	}
	fmt.Printf("gray: %d bytes crc=%08x\n", gb.Len(), crc32.ChecksumIEEE(gb.Bytes()))
	gm, err := jpeg.Decode(bytes.NewReader(gb.Bytes()))
	fmt.Println("gray decode:", err, gm.Bounds(), gm.At(1, 1))

	// The errors.
	_, err = jpeg.Decode(bytes.NewReader([]byte("not a jpeg")))
	fmt.Println("bad header:", err)
	_, err = jpeg.Decode(bytes.NewReader(raw[:20]))
	fmt.Println("short read:", err)

	var fe jpeg.FormatError = "bad chunk"
	fmt.Println("format error:", fe.Error())
	var ue jpeg.UnsupportedError = "odd feature"
	fmt.Println("unsupported error:", ue.Error())
	fmt.Println("quality:", jpeg.DefaultQuality)
}
