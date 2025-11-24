package pipeline

import (
	"log/slog"
	"time"
	"webcam/internal/camera"
	"webcam/internal/filters"

	"gocv.io/x/gocv"
)

// ReadStage reads frames from a camera and sends them to the pipeline.
func ReadStage(cam *camera.Camera, done <-chan struct{}) <-chan Frame {
	out := make(chan Frame)

	go func() {
		defer close(out)

		frame := gocv.NewMat()
		defer frame.Close()

		for {
			select {
			case <-done:
				return
			default:
				if ok := cam.Read(&frame); !ok || frame.Empty() {
					continue
				}

				out <- Frame{
					Img:  frame.Clone(),
					Time: time.Now(),
				}
			}
		}
	}()

	return out
}

// mapErr logs stage errors.
func mapErr(stage string, err error) {
	if err != nil {
		slog.Error("stage error", "stage", stage, "err", err)
	}
}

// GrayStage converts frames to grayscale.
func GrayStage() StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.Gray(f.Img)
					f.Close()
					if err != nil {
						mapErr("gray", err)
						continue
					}
					next <- Frame{Img: dst, Time: time.Now()}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// BlurStage applies Gaussian blur.
func BlurStage(ksize int) StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.Blur(f.Img, ksize)
					f.Close()
					if err != nil {
						mapErr("blur", err)
						continue
					}
					next <- Frame{Img: dst, Time: time.Now()}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// EdgeStage detects edges using the Canny algorithm.
func EdgeStage(th1, th2 float32) StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.Edge(f.Img, th1, th2)
					f.Close()
					if err != nil {
						mapErr("edge", err)
						continue
					}
					next <- Frame{Img: dst, Time: time.Now()}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// BrightnessContrastStage adjusts brightness and contrast.
func BrightnessContrastStage(alpha, beta float64) StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.BrightnessContrast(f.Img, alpha, beta)
					f.Close()
					if err != nil {
						mapErr("brightness_contrast", err)
						continue
					}
					next <- Frame{Img: dst, Time: time.Now()}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// SharpenStage increases image sharpness.
func SharpenStage() StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.Sharpen(f.Img)
					f.Close()
					if err != nil {
						mapErr("sharpen", err)
						continue
					}
					next <- Frame{Img: dst, Time: time.Now()}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// PassthroughStage forwards frames without modification.
func PassthroughStage() StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					next <- f
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// CloneStage a protective copy of the frame.
func CloneStage() StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					cl := f.Img.Clone()
					f.Close()
					next <- Frame{Img: cl, Time: time.Now()}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// ToBGRStage an envelope from GRAY to BGR (for UI compatibility).
func ToBGRStage() StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}) <-chan Frame {
		next := make(chan Frame)
		go func() {
			defer close(next)
			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					if f.Img.Type() == gocv.MatTypeCV8UC1 {
						dst := gocv.NewMat()
						_ = gocv.CvtColor(f.Img, &dst, gocv.ColorGrayToBGR)
						f.Close()
						next <- Frame{Img: dst, Time: time.Now()}
						continue
					}
					next <- f
				case <-done:
					return
				}
			}
		}()
		return next
	}
}
