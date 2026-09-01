// cmd_genicon — генерирует icon.png и icon.ico приложения shortTxt.
// Временная утилита: запускается вручную, результат хранится в репозитории.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"

	"golang.org/x/image/draw"
	"golang.org/x/image/vector"
)

const size = 256

var (
	bgColor    = color.NRGBA{R: 63, G: 81, B: 181, A: 255}   // индиго
	textColor  = color.NRGBA{R: 255, G: 255, B: 255, A: 255} // белый T
	caretColor = color.NRGBA{R: 255, G: 171, B: 64, A: 255}  // янтарный курсор
)

func main() {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	// фон — скруглённый квадрат
	fillRoundRect(img, 8, 8, 240, 240, 44, bgColor)

	// стилизованная «T» — перекладина и стойка
	fillRoundRect(img, 52, 64, 124, 26, 10, textColor)
	fillRoundRect(img, 100, 90, 28, 108, 10, textColor)

	// текстовый курсор справа
	fillRoundRect(img, 160, 96, 22, 128, 9, caretColor)

	pngFile, err := os.Create("icon.png")
	must(err)
	must(png.Encode(pngFile, img))
	must(pngFile.Close())

	must(os.WriteFile("icon.ico", buildICO(img), 0o644))
}

func fillRoundRect(img *image.NRGBA, x, y, w, h, r int, c color.NRGBA) {
	ras := vector.NewRasterizer(size, size)
	f := func(v int) float32 { return float32(v) }
	k := float32(0.45)

	// все координаты сразу во float32
	x0, y0 := f(x), f(y)
	x1, y1 := f(x+w), f(y+h)
	xr, yr := x1-f(r), y1-f(r)

	ras.MoveTo(x0+f(r), y0)
	ras.LineTo(xr, y0)
	ras.CubeTo(x1-f(r)*k, y0, x1, y0+f(r)*k, x1, y0+f(r))
	ras.LineTo(x1, yr)
	ras.CubeTo(x1, y1-f(r)*k, x1-f(r)*k, y1, xr, y1)
	ras.LineTo(x0+f(r), y1)
	ras.CubeTo(x0+f(r)*k, y1, x0, y1-f(r)*k, x0, yr)
	ras.LineTo(x0, y0+f(r))
	ras.CubeTo(x0, y0+f(r)*k, x0+f(r)*k, y0, x0+f(r), y0)
	ras.Draw(img, image.Rectangle{Max: image.Point{X: size, Y: size}},
		image.NewUniform(c), image.Point{})
}

// buildICO собирает ICO с изображениями 16/32/48/256 (PNG-упаковка).
func buildICO(src *image.NRGBA) []byte {
	sizes := []int{16, 32, 48, 256}

	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 1, 0})
	binary.Write(&buf, binary.LittleEndian, uint16(len(sizes)))

	var entries []byte
	var blobs [][]byte
	offset := 6 + 16*len(sizes)
	for _, s := range sizes {
		dst := image.NewNRGBA(image.Rect(0, 0, s, s))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		var pngBuf bytes.Buffer
		must(png.Encode(&pngBuf, dst))
		blobs = append(blobs, pngBuf.Bytes())

		entry := make([]byte, 16)
		if s == 256 {
			entry[0], entry[1] = 0, 0 // 256 кодируется нулём
		} else {
			entry[0], entry[1] = byte(s), byte(s)
		}
		binary.LittleEndian.PutUint16(entry[4:], 1)  // planes
		binary.LittleEndian.PutUint16(entry[6:], 32) // bitcount
		binary.LittleEndian.PutUint32(entry[8:], uint32(pngBuf.Len()))
		binary.LittleEndian.PutUint32(entry[12:], uint32(offset))
		entries = append(entries, entry...)
		offset += pngBuf.Len()
	}
	buf.Write(entries)
	for _, b := range blobs {
		buf.Write(b)
	}
	return buf.Bytes()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
