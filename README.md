# Webcam noise and stability

A desktop application that measures the noise and stability of a webcam. It captures video from the camera, computes the noise of every frame and saves a report with the results. The interface is available in English and Russian; the language is chosen at start.

## Features

- **Four experiment modes**
  - **Dark:** noise floor of the camera and dark signal non-uniformity (DSNU).
  - **Light:** noise under even light and photo response non-uniformity (PRNU).
  - **Stability:** drift of the level and the noise over a long run.
  - **Modulator:** signal and signal-to-noise ratio of a light source switched on and off.
- **Mode matrix:** runs the same experiment in several camera modes (resolution, frame rate, pixel format) and puts the results into one comparison table.
- **Camera mode probe:** measures the frame rate the camera really delivers in every mode, instead of trusting the driver.
- **Real-time noise graph** of the temporal noise and the level over the last 60 seconds.
- **Auto controls lock:** turns off auto exposure, white balance and focus, so the camera does not change the signal during a measurement.
- **Reports:** a text report, per-frame CSV data that opens in Excel, the average frame, anoise map and a defect pixel map.

## Documentation

All documentation is in the [docs](docs/README.md) folder:

| | English | Русский |
|---|---|---|
| User guide: modes, how to use them and what they give you | [guide](docs/en/guide.md) | [руководство](docs/ru/guide.md) |
| Glossary: all terms and CSV columns | [glossary](docs/en/glossary.md) | [глоссарий](docs/ru/glossary.md) |

## Getting the program

### Windows

Every push to `develop` and `main` builds the program in GitHub Actions. Download the `webcam-windows-x64` artifact from the latest successful run of the **Build Windows** workflow, unpack it and run `webcam.exe`. The artifact contains the required DLLs and the `docs` folder.

### macOS (from source)

Requirements: Go (see `go.mod` for the version), Xcode command line tools and OpenCV from Homebrew:

```sh
brew install opencv
make run            # run the application
make test           # run the tests (camera hardware tests are skipped)
make build          # build build/webcam-macos for Apple Silicon
```

On macOS the auto controls lock is not available: OpenCV's AVFoundation backend does not give access to exposure, white balance and focus.

## Output

All results are written to the folder the program is started from:

| Folder | Contents |
|---|---|
| `experiments/` | One folder per experiment or mode matrix: report, CSV data and maps |
| `noise_log/` | Data of the real-time noise graph |
| `screenshots/`, `videos/` | Screenshots and video recordings |
| `cache/` | Saved camera mode measurements |
| `logs/` | Technical log of the program |

## Built with

- [Go](https://go.dev/)
- [GoCV](https://gocv.io/) and [OpenCV](https://opencv.org/) for camera capture and image processing
- [Go fonts](https://go.dev/blog/go-fonts) for on-screen text with Cyrillic support
