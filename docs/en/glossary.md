# Glossary

Terms used in the program, the reports and the tables. How to use the program is described in
the [user guide](guide.md).

## Contents

- [Basics](#basics)
- [Noise and stability over time](#noise-and-stability-over-time)
- [Non-uniformity across the field](#non-uniformity-across-the-field)
- [Modulator](#modulator)
- [Camera and capture modes](#camera-and-capture-modes)
- [Camera auto controls](#camera-auto-controls)
- [Maps and pictures](#maps-and-pictures)
- [CSV columns](#csv-columns)

## Basics

**DN (digital number).** The unit of pixel brightness. The program converts a frame to
grayscale, where every pixel has a value from 0 (black) to 255 (white). 1 DN is one step of
this scale. All levels and noise values in the reports are in DN.

**Level (mean).** The average brightness of a frame or an area, in DN. It is near zero in the
dark and high on a bright scene.

**Stddev (standard deviation).** A measure of how much values spread around their average. The
larger the stddev, the more the values "jump". Almost all kinds of noise in the program are
computed with it.

**Noise.** Random fluctuations of a pixel value that are not in the scene itself. The lower the
noise, the fainter the details and light changes the camera can tell apart.

**Noise threshold.** A temporal noise value in DN that you set yourself (keys −/=, 0.1 to
50 DN, default 2). The threshold does not affect the measurements. The program only marks the
frames and pixels whose noise is above it.

**Above threshold.** How many frames (or pixels) had noise above the threshold, as a count and a
percentage.

**Run, experiment.** A recording of the selected duration (30 s – 30 min) during which all
values are computed and a report is saved.

**Experiment mode.** The measurement scenario: dark, light, stability or modulator. It defines
how to prepare the camera and how to read the results. See the
[user guide](guide.md#experiment-modes).

## Noise and stability over time

**Temporal noise.** The main noise figure, computed for every frame: the difference between the
current and the previous frame is taken, its stddev over all pixels is computed and divided by
√2. The division by √2 is needed because the difference holds the noise of two frames. When
the whole picture becomes evenly brighter or darker (lamp flicker), the subtraction removes that
change, so it does not count as noise.

**Pixel noise.** The stddev of every single pixel over the whole run, then averaged over the
frame (and its maximum). Unlike temporal noise, it includes light flicker and slow drift.

**Spatial spread (spatial_std).** The stddev of pixel values within one frame. It includes
noise, scene content and uneven lighting.

**Drift.** How much the level of the last frame differs from the first one, in DN. It shows
whether the brightness "wanders" over time.

**Trend.** The rate of level change in DN per minute, fitted over all frames (linear
regression). Less sensitive to random jumps than drift.

**Late frames.** Frames that came more than 1.5 times later than the nominal interval. More
than 5% of them means the computer could not keep up with the stream.

**Frame interval.** The time between neighbouring frames: average, minimum and maximum, in
milliseconds.

## Non-uniformity across the field

These are fixed, non-random differences between pixels: a "pattern" of the sensor that repeats
in every frame.

**Non-uniformity.** The stddev of per-pixel averages over the run. It would be zero if all
pixels were identical. What it is called depends on the mode:

- **DSNU (dark signal non-uniformity):** non-uniformity in the dark (dark mode, "off" frames
  of the modulator). It shows how differently pixels "glow" without light.
- **PRNU (photo response non-uniformity):** non-uniformity under even light (light mode, "on"
  frames of the modulator). It shows how differently pixels respond to the same light. It is
  also given as a percentage of the level.
- In the stability mode the non-uniformity includes the scene content, so compare it only
  between runs of the same scene.

**Row and column FPN (fixed pattern noise).** The stddev of row (or column) averages. A large
value means horizontal (or vertical) stripes in the frame.

**8×6 zones.** The frame is split into 8 columns and 6 rows of zones. For each zone the average
level and noise are given ("level/noise"). This shows, for example, darker corners (vignetting)
or a noisy corner. The zone spread is the lowest and highest zone level.

**Never-changing pixels.** Pixels whose stddev over the run is below 0.1 DN: dead or stuck
pixels, or pixels clipped at 0 or 255. A large share of them in the dark means the camera clips
the black level and dark noise is underestimated.

**Noisy pixels.** Pixels whose noise is more than 3 times the average noise.

**Saturated pixels.** Pixels with an average level of about 250 DN or more. They hit the top of
the scale, so their noise is underestimated.

## Modulator

**Modulator.** The mode in which the light is switched on and off periodically. The program
detects the light state from the average frame brightness.

**Light state: on / off / transition / unknown.** "Unknown" means the program has not seen a
difference between bright and dark frames yet. "Transition" means the light is switching or the
level has not settled. Such frames, and 2 frames right after every switch, are not used.

**Hysteresis.** So that noise does not cause false switches, the state changes only when the
level crosses the middle between on and off with a margin of ±20% of their difference.

**On/off contrast.** The difference between the on and off levels. Below 5 DN the light change is
not detected.

**Signal.** The average level of "on" frames minus the average level of "off" frames, in DN.

**SNR (signal-to-noise ratio).** The signal divided by the temporal noise of the "on" frames. It
shows how many times the useful change is larger than the random noise. SNR = 1 means the signal
is lost in the noise. The larger the SNR, the more reliable the detection.

**dB (decibels).** SNR on a logarithmic scale: 20·log₁₀(SNR). An SNR of 10 is 20 dB, an SNR of
100 is 40 dB.

**Switches, segments, period.** The number of state changes, the length of the on and off
segments and the full cycle period (on + off) in seconds. The frequency in Hz is 1 / period.

## Camera and capture modes

**Resolution.** The frame size in pixels, e.g. 1280×720.

**FPS (frames per second).** The frame rate. The program uses several:

- **requested:** what the program asked the camera for;
- **driver:** what the driver reports. It often just repeats the request;
- **measured:** frames really received while measuring the modes;
- **effective:** the real rate during an experiment.

**Confirmed / not confirmed.** A frame rate is confirmed when the measured rate is within 20% of
the requested one. The camera does not actually deliver frame rates that are not confirmed in
that mode, so they are not offered for selection. "Size changed" means the driver changed the
resolution by itself to deliver that frame rate.

**Pixel format (FourCC).** How the camera transfers frames:

- **MJPG:** JPEG-compressed. It allows high resolutions and frame rates but adds compression
  artifacts that affect the noise.
- **YUY2** and **NV12:** uncompressed. Noise is measured more faithfully, but frame rates are
  lower at high resolutions.
- **auto:** the driver chooses.

**Backend.** How the program (through the OpenCV library) accesses the camera:

- **Media Foundation (MSMF):** the standard one on Windows.
- **DirectShow:** older, on Windows, but it allows control of the auto settings and opens the
  driver settings dialog.
- **AVFoundation:** on macOS. It does not allow control of exposure and focus.

**Mode measurement (probe).** Going through resolutions and frame rates at start and measuring
the real frame rate. The result is saved in the `cache/` folder.

**Warm-up.** In the mode matrix, 3 seconds after a camera mode switch before the measurement
starts, so the frame stream can settle.

**Mode matrix.** The same experiment run automatically in several camera modes one after another,
with a common results table.

## Camera auto controls

**Auto exposure.** The camera changes the exposure time and gain to match the light. Turn it off
for measurements, otherwise level and noise change because of the automation, not the sensor.

**Exposure.** The time light is collected for one frame. The longer, the brighter the frame.

**Gain.** Electronic amplification of the signal. It raises the brightness, and the noise with
it.

**White balance (WB).** Colour correction for the type of light. Auto white balance changes the
colour proportions, and with them the grayscale brightness.

**Auto focus.** The camera focuses by itself. Refocusing changes the picture between frames.

**Auto controls lock (key L).** The program turns off auto exposure, auto white balance and auto
focus and keeps their current values. The report says whether they were locked.

## Maps and pictures

**mean.png (average frame).** Every pixel averaged over the run. Random noise disappears and the
fixed pattern remains: non-uniformity, stripes, vignetting.

**noise_map.png (noise map).** The brightness of each point shows the noise of that pixel: the
brighter, the noisier.

**defects.png (defect map).** Blue points are pixels that never change. Red points are very
noisy pixels.

**_on / _off suffixes.** In the modulator mode the pictures are built separately for the light-on
and light-off frames.

## CSV columns

All CSV files use the `;` separator and a decimal comma and open in Excel with a double click.

### data.csv (experiment, per frame)

| Column | Meaning |
|---|---|
| `frame` | frame number |
| `t_sec` | time from the start of the run, s |
| `mean` | frame level, DN |
| `drift` | level minus the level of the first frame, DN |
| `spatial_std` | spatial spread, DN |
| `temporal_noise` | temporal noise, DN (empty for the first frame) |
| `above_threshold` | 1 when the noise is above the threshold, otherwise 0 |
| `state` | modulator light state: `on`, `off`, `transition`, `unknown` |

### summary.csv (mode matrix, one mode per row)

| Column | Meaning |
|---|---|
| `requested` | the requested mode, e.g. `1280x720@30 MJPG` |
| `width`, `height` | the actual resolution |
| `format` | the actual pixel format |
| `fps_driver`, `fps_effective` | FPS reported by the driver and the effective FPS |
| `frames` | number of frames |
| `level` | average level, DN |
| `temporal_noise` | average temporal noise, DN |
| `above_threshold_pct` | share of frames above the threshold, % |
| `pixel_noise` | average pixel noise, DN |
| `non_uniformity` | non-uniformity (DSNU/PRNU), DN |
| `row_fpn`, `col_fpn` | row and column FPN, DN |
| `drift` | drift, DN |
| `off_level`, `off_noise` | modulator: level and noise of the "off" frames |
| `signal`, `snr` | modulator: signal and SNR |
| `switches`, `period_s` | modulator: number of switches and period, s |
| `dir` | folder with the detailed report of this mode |

In `summary.txt` the same values have short headers: t.noise = temporal noise, px.noise = pixel
noise, nonunif = non-uniformity, eff.fps = effective FPS.

### noise_log/noise_*.csv (noise graph window)

| Column | Meaning |
|---|---|
| `time` | frame time (hours:minutes:seconds.ms) |
| `t_sec` | time since the window was opened, s |
| `mean` | level, DN |
| `temporal_noise` | temporal noise, DN |
| `threshold` | threshold at that moment, DN |
| `above_threshold` | 1 when the noise is above the threshold |
