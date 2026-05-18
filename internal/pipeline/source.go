package pipeline

import (
	"log/slog"
	"sync"
	"time"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/filters"

	"gocv.io/x/gocv"
)

// ReadStage reads frames from a camera and sends them to the pipeline.
func ReadStage(cam *camera.Camera, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
	out := make(chan Frame)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(out)

		frame := gocv.NewMat()
		defer frame.Close()

		for {
			select {
			case <-done:
				return
			default:
			}

			if ok := cam.Read(&frame); !ok || frame.Empty() {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			cloned := frame.Clone()

			select {
			case <-done:
				cloned.Close()
				return
			case out <- Frame{
				Img:  cloned,
				Time: time.Now(),
			}:
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
	return func(prev <-chan Frame, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
		next := make(chan Frame)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(next)

			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					if f.Img.Channels() == 1 {
						select {
						case <-done:
							f.Close()
							return
						case next <- f:
						}
						continue
					}
					dst, err := filters.Gray(f.Img)
					f.Close()
					if err != nil {
						mapErr("gray", err)
						continue
					}

					select {
					case <-done:
						dst.Close()
						return
					case next <- Frame{
						Img:  dst,
						Time: time.Now(),
					}:
					}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// BlurStage applies Gaussian blur.
func BlurStage(p *control.BlurParams) StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
		next := make(chan Frame)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(next)

			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.Blur(f.Img, p.Ksize)
					f.Close()
					if err != nil {
						mapErr("blur", err)
						continue
					}

					select {
					case <-done:
						dst.Close()
						return
					case next <- Frame{
						Img:  dst,
						Time: time.Now(),
					}:
					}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// EdgeStage detects edges using the Canny algorithm.
func EdgeStage(p *control.EdgeParams) StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
		next := make(chan Frame)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(next)

			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.Edge(f.Img, p.Threshold1, p.Threshold2)
					f.Close()
					if err != nil {
						mapErr("edge", err)
						continue
					}

					select {
					case <-done:
						dst.Close()
						return
					case next <- Frame{
						Img:  dst,
						Time: time.Now(),
					}:
					}
				case <-done:
					return
				}
			}
		}()
		return next
	}
}

// BrightnessContrastStage adjusts brightness and contrast.
func BrightnessContrastStage(p *control.BrightnessContrastParams) StageFunc {
	return func(prev <-chan Frame, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
		next := make(chan Frame)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(next)

			for {
				select {
				case f, ok := <-prev:
					if !ok {
						return
					}
					dst, err := filters.BrightnessContrast(f.Img, p.Alpha, p.Beta)
					f.Close()
					if err != nil {
						mapErr("brightness_contrast", err)
						continue
					}

					select {
					case <-done:
						dst.Close()
						return
					case next <- Frame{
						Img:  dst,
						Time: time.Now(),
					}:
					}
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
	return func(prev <-chan Frame, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
		next := make(chan Frame)

		wg.Add(1)
		go func() {
			defer wg.Done()
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

					select {
					case <-done:
						dst.Close()
						return
					case next <- Frame{
						Img:  dst,
						Time: time.Now(),
					}:
					}
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
	return func(prev <-chan Frame, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
		next := make(chan Frame)

		wg.Add(1)
		go func() {
			defer wg.Done()
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

						select {
						case <-done:
							dst.Close()
							return
						case next <- Frame{
							Img:  dst,
							Time: time.Now(),
						}:
						}
						continue
					}

					select {
					case <-done:
						f.Close()
						return
					case next <- f:
					}

				case <-done:
					return
				}
			}
		}()
		return next
	}
}
