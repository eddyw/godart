// The program proves that image/png writes an image and reads it again. The
// encoder converts a pointer to encoder to a pointer to EncoderBuffer and back.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

type pool struct {
	held *png.EncoderBuffer
	gets int
}

func (p *pool) Get() *png.EncoderBuffer {
	p.gets++
	return p.held
}

func (p *pool) Put(b *png.EncoderBuffer) {
	p.held = b
}

func main() {
	src := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for y := range 3 {
		for x := range 4 {
			src.Set(x, y, color.RGBA{R: uint8(x * 60), G: uint8(y * 80), B: 7, A: 255})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, src); err != nil {
		panic(err)
	}
	fmt.Println("signature", out.Bytes()[1] == 'P', out.Bytes()[2] == 'N', out.Bytes()[3] == 'G')

	back, err := png.Decode(bytes.NewReader(out.Bytes()))
	if err != nil {
		panic(err)
	}
	fmt.Println("bounds", back.Bounds())
	for y := range 3 {
		for x := range 4 {
			r, g, b, a := back.At(x, y).RGBA()
			fmt.Printf("%d,%d=%d/%d/%d/%d ", x, y, r>>8, g>>8, b>>8, a>>8)
		}
	}
	fmt.Println()

	cfg, err := png.DecodeConfig(bytes.NewReader(out.Bytes()))
	if err != nil {
		panic(err)
	}
	fmt.Println("config", cfg.Width, cfg.Height, cfg.ColorModel == color.NRGBAModel)

	p := &pool{}
	enc := &png.Encoder{CompressionLevel: png.BestCompression, BufferPool: p}
	var second bytes.Buffer
	if err := enc.Encode(&second, src); err != nil {
		panic(err)
	}
	var third bytes.Buffer
	if err := enc.Encode(&third, src); err != nil {
		panic(err)
	}
	fmt.Println("pool gets", p.gets, "buffer held", p.held != nil)
	fmt.Println("2 runs agree", bytes.Equal(second.Bytes(), third.Bytes()))

	if _, err := png.Decode(bytes.NewReader([]byte("not a png"))); err == nil {
		panic("a wrong header must give an error")
	} else {
		fmt.Println("wrong header rejected")
	}
}
