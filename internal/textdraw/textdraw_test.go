package textdraw

import (
	"image/color"
	"testing"

	"gocv.io/x/gocv"
)

func TestDrawCyrillic(t *testing.T) {
	img := gocv.NewMatWithSize(60, 300, gocv.MatTypeCV8UC3)
	defer img.Close()

	Draw(&img, "Шум 1.5 DN", 10, 40, 24, 2, color.RGBA{255, 255, 255, 0})

	white := 0
	for y := 0; y < img.Rows(); y++ {
		for x := 0; x < img.Cols(); x++ {
			v := img.GetVecbAt(y, x)
			if v[0] == 255 && v[1] == 255 && v[2] == 255 {
				white++
			}
		}
	}
	// the background is black already, so check the letters were painted
	if white == 0 {
		t.Fatal("no text pixels drawn")
	}
}

func TestWidthGrowsWithText(t *testing.T) {
	if a, b := Width("Шум", 20), Width("Шум шум", 20); a <= 0 || b <= a {
		t.Errorf("Width = %d, %d; want positive and growing", a, b)
	}
}

func TestDrawClipsAtEdges(t *testing.T) {
	img := gocv.NewMatWithSize(20, 20, gocv.MatTypeCV8UC1)
	defer img.Close()
	Draw(&img, "Текст за краем кадра", -5, 30, 30, 3, color.RGBA{255, 0, 0, 0}) // must not panic
}
