// Program gifuse writes a GIF image and reads it again. It shows the encoder,
// the decoder, and the animation of image/gif.
package main

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
)

// frame builds a paletted image with a diagonal line.
func frame(w, h int, pal color.Palette, index uint8) *image.Paletted {
	m := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	for i := 0; i < w && i < h; i++ {
		m.SetColorIndex(i, i, index)
	}
	return m
}

func main() {
	pal := color.Palette{
		color.RGBA{0, 0, 0, 255},
		color.RGBA{255, 0, 0, 255},
		color.RGBA{0, 255, 0, 255},
		color.RGBA{0, 0, 255, 255},
	}

	// 1 image.
	one := frame(8, 6, pal, 1)
	var b bytes.Buffer
	if err := gif.Encode(&b, one, &gif.Options{NumColors: 4}); err != nil {
		fmt.Println("encode:", err)
		return
	}
	raw := b.Bytes()
	fmt.Printf("encode: %d bytes crc=%08x head=%q\n", len(raw),
		crc32.ChecksumIEEE(raw), raw[:6])

	m, err := gif.Decode(bytes.NewReader(raw))
	fmt.Println("decode err:", err, "bounds:", m.Bounds())
	pm := m.(*image.Paletted)
	fmt.Println("pixel:", pm.ColorIndexAt(0, 0), pm.ColorIndexAt(1, 1), pm.At(2, 2))
	fmt.Println("palette:", len(pm.Palette))

	cfg, err := gif.DecodeConfig(bytes.NewReader(raw))
	fmt.Println("config:", cfg.Width, cfg.Height, err)

	// The registry of image.Decode reads the same bytes.
	m2, name, err := image.Decode(bytes.NewReader(raw))
	fmt.Println("registry:", name, err, m2.Bounds())
	cfg2, name2, err := image.DecodeConfig(bytes.NewReader(raw))
	fmt.Println("registry config:", name2, cfg2.Width, cfg2.Height, err)

	// An animation of 3 frames.
	all := &gif.GIF{
		Image:     []*image.Paletted{frame(8, 6, pal, 1), frame(8, 6, pal, 2), frame(8, 6, pal, 3)},
		Delay:     []int{10, 20, 30},
		LoopCount: 2,
		Disposal:  []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalPrevious},
	}
	var ab bytes.Buffer
	if err := gif.EncodeAll(&ab, all); err != nil {
		fmt.Println("encode all:", err)
		return
	}
	araw := ab.Bytes()
	fmt.Printf("encode all: %d bytes crc=%08x\n", len(araw), crc32.ChecksumIEEE(araw))

	back, err := gif.DecodeAll(bytes.NewReader(araw))
	fmt.Println("decode all err:", err)
	fmt.Println("frames:", len(back.Image), "delay:", back.Delay,
		"loop:", back.LoopCount, "disposal:", back.Disposal)
	fmt.Println("config:", back.Config.Width, back.Config.Height,
		back.BackgroundIndex, len(back.Config.ColorModel.(color.Palette)))
	for i, f := range back.Image {
		fmt.Printf("frame %d bounds=%v first=%d\n", i, f.Bounds(), f.ColorIndexAt(0, 0))
	}

	// The errors.
	_, err = gif.Decode(bytes.NewReader([]byte("not a gif at all")))
	fmt.Println("bad header:", err)
	_, err = gif.Decode(bytes.NewReader(raw[:10]))
	fmt.Println("short read:", err)

	tooMany := image.NewPaletted(image.Rect(0, 0, 2, 2), pal)
	err = gif.Encode(&bytes.Buffer{}, tooMany, &gif.Options{NumColors: 1})
	fmt.Println("1 color:", err)

	big := image.NewPaletted(image.Rect(0, 0, 70000, 1), pal)
	err = gif.Encode(&bytes.Buffer{}, big, nil)
	fmt.Println("too wide:", err)

	fmt.Println("disposal:", gif.DisposalNone, gif.DisposalBackground, gif.DisposalPrevious)

}
