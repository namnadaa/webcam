package media

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gocv.io/x/gocv"
)

// RecorderEvent represents recorder state changes.
type RecorderEvent int

const (
	RecorderNone RecorderEvent = iota
	RecorderStarted
	RecorderStopped
)

// VideoRecorder handles video recording using OpenCV VideoWriter.
type VideoRecorder struct {
	writer    *gocv.VideoWriter
	recording bool
}

// Start initializes video recording and opens output file.
func (v *VideoRecorder) Start(filename string, fps float64, width, height int) {
	if v.recording {
		return
	}

	w, err := gocv.VideoWriterFile(filename, "MJPG", fps, width, height, true)
	if err != nil {
		slog.Error("video start error", "err", err)
		return
	}

	if w == nil {
		slog.Warn("VideoWriter is NIL")
		return
	}

	v.writer = w
	v.recording = true

	slog.Info("video recording started", "file", filename)
}

// Write writes a single frame into the video stream.
func (v *VideoRecorder) Write(frame gocv.Mat) {
	if !v.recording || v.writer == nil {
		return
	}

	v.writer.Write(frame)
}

// Stop finalizes recording and releases VideoWriter resources.
func (v *VideoRecorder) Stop() {
	if !v.recording || v.writer == nil {
		return
	}

	v.writer.Close()
	v.writer = nil
	v.recording = false

	slog.Info("video recording stopped")
}

// IsRecording returns current recording state.
func (v *VideoRecorder) IsRecording() bool {
	return v.recording
}

// UpdateRecorder controls recording lifecycle based on state.
func UpdateRecorder(recorder *VideoRecorder, state *State, fps float64, width, height int) RecorderEvent {
	safeFPS := fps
	if safeFPS <= 1 {
		safeFPS = 30
	}

	if state.Recording && !recorder.IsRecording() {
		err := os.MkdirAll("videos", 0755)
		if err != nil {
			slog.Error("video dir error", "err", err)
			return RecorderNone
		}

		filename := fmt.Sprintf(
			"video_%s.avi",
			time.Now().Format("2006-01-02_15-04-05"),
		)

		path := filepath.Join("videos", filename)

		recorder.Start(path, safeFPS, width, height)
		return RecorderStarted
	}

	if !state.Recording && recorder.IsRecording() {
		recorder.Stop()
		return RecorderStopped
	}

	return RecorderNone
}
