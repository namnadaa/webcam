// Package textdraw renders text on OpenCV images with a TrueType font.
// OpenCV's own Hershey fonts have Latin letters only, so Cyrillic text needs this.
package textdraw

import (
	"image"
	"image/color"
	"sync"

	"gocv.io/x/gocv"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// maxCached is how many rendered texts are kept; changing numbers produce new texts every frame,
// so the cache is simply dropped when it is full.
const maxCached = 1024

// mask is a rendered text: coverage of the letters and of the outline around them.
type mask struct {
	text, outline *image.Alpha
	ascent        int // pixels from the top of the mask to the baseline
}

type cacheKey struct {
	text    string
	size    int
	outline int
}

var (
	mu     sync.Mutex
	parsed *opentype.Font
	faces  = map[int]font.Face{}
	masks  = map[cacheKey]*mask{}
)

// face returns the font face of the given pixel size.
func face(size int) font.Face {
	if f, ok := faces[size]; ok {
		return f
	}

	if parsed == nil {
		f, err := opentype.Parse(gomono.TTF)
		if err != nil {
			panic("textdraw: embedded font: " + err.Error())
		}
		parsed = f
	}

	f, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic("textdraw: font face: " + err.Error())
	}
	faces[size] = f
	return f
}

// Width returns the width of text in pixels at the given font size.
func Width(text string, size int) int {
	mu.Lock()
	defer mu.Unlock()
	return font.MeasureString(face(size), text).Ceil()
}

// Draw renders text with its baseline starting at (x, y) in the color clr,
// surrounded by a black outline of the given width (0 = none) for readability on any background.
func Draw(img *gocv.Mat, text string, x, y, size, outline int, clr color.RGBA) {
	if text == "" || img.Empty() || !img.IsContinuous() {
		return
	}

	mu.Lock()
	m := render(text, size, outline)
	mu.Unlock()

	left, top := x-outline, y-m.ascent
	if m.outline != nil {
		blend(img, m.outline, left, top, color.RGBA{})
	}
	blend(img, m.text, left, top, clr)
}

// render returns the cached mask of text or renders it.
func render(text string, size, outline int) *mask {
	key := cacheKey{text, size, outline}
	if m, ok := masks[key]; ok {
		return m
	}
	if len(masks) >= maxCached {
		masks = map[cacheKey]*mask{}
	}

	f := face(size)
	metrics := f.Metrics()
	ascent := metrics.Ascent.Ceil() + outline
	w := font.MeasureString(f, text).Ceil() + 2*outline
	h := ascent + metrics.Descent.Ceil() + outline

	m := &mask{text: image.NewAlpha(image.Rect(0, 0, w, h)), ascent: ascent}
	d := font.Drawer{Dst: m.text, Src: image.Opaque, Face: f, Dot: fixed.P(outline, ascent)}
	d.DrawString(text)

	if outline > 0 {
		m.outline = dilate(m.text, outline)
	}

	masks[key] = m
	return m
}

// dilate spreads the coverage of src by r pixels in every direction:
// a horizontal and then a vertical max pass (a square brush).
func dilate(src *image.Alpha, r int) *image.Alpha {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	tmp := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		row := src.Pix[y*src.Stride : y*src.Stride+w]
		for x := 0; x < w; x++ {
			var v uint8
			for dx := max(0, x-r); dx <= min(w-1, x+r); dx++ {
				v = max(v, row[dx])
			}
			tmp[y*w+x] = v
		}
	}

	dst := image.NewAlpha(b)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var v uint8
			for dy := max(0, y-r); dy <= min(h-1, y+r); dy++ {
				v = max(v, tmp[dy*w+x])
			}
			dst.Pix[y*dst.Stride+x] = v
		}
	}
	return dst
}

// blend paints clr through the coverage mask onto the image with its top-left corner at (left, top).
// The image is BGR or grayscale.
func blend(img *gocv.Mat, m *image.Alpha, left, top int, clr color.RGBA) {
	data, err := img.DataPtrUint8()
	if err != nil {
		return
	}

	channels := img.Channels()
	cols, rows := img.Cols(), img.Rows()
	paint := []uint8{clr.B, clr.G, clr.R}
	if channels == 1 {
		paint = []uint8{uint8((299*int(clr.R) + 587*int(clr.G) + 114*int(clr.B)) / 1000)}
	}

	b := m.Bounds()
	for my := 0; my < b.Dy(); my++ {
		y := top + my
		if y < 0 || y >= rows {
			continue
		}
		for mx := 0; mx < b.Dx(); mx++ {
			x := left + mx
			if x < 0 || x >= cols {
				continue
			}
			a := int(m.Pix[my*m.Stride+mx])
			if a == 0 {
				continue
			}

			p := (y*cols + x) * channels
			for c := 0; c < len(paint) && c < channels; c++ {
				data[p+c] = uint8((int(paint[c])*a + int(data[p+c])*(255-a)) / 255)
			}
		}
	}
}
