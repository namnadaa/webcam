package media

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gocv.io/x/gocv"
)

// SaveScreenshot saves current frame as image.
func SaveScreenshot(img gocv.Mat) error {
	err := os.MkdirAll("screenshots", 0755)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf(
		"screenshot_%s.jpg",
		time.Now().Format("2006-01-02_15-04-05"),
	)

	path := filepath.Join("screenshots", filename)

	ok := gocv.IMWrite(path, img)
	if !ok {
		return fmt.Errorf("failed to save screenshot")
	}

	return nil
}
