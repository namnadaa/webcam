package media

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gocv.io/x/gocv"
)

// SaveScreenshot saves current frame as image.
func SaveScreenshot(img gocv.Mat) error {
	err := os.MkdirAll("screenshots", 0755)
	if err != nil {
		slog.Error("screenshots dir error", "err", err)
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

// HandleScreenshot saves frame if screenshot flag is set.
func HandleScreenshot(state *State, frame gocv.Mat) bool {
	if !state.Screenshot {
		return false
	}

	err := SaveScreenshot(frame)
	if err != nil {
		slog.Error("screenshots error", "err", err)
		state.Screenshot = false
		return false

	}
	slog.Info("screenshot saved")
	state.Screenshot = false

	return true
}
