# User guide

The program measures the noise and stability of a webcam. It captures video from the camera, computes the noise of every frame and saves a report: how noisy the camera is, how evenly it renders the field of view and whether the picture changes over time.

Every term (DN, temporal noise, DSNU, PRNU, FPN, SNR and so on) is explained in the [glossary](glossary.md).

## Contents

1. [What the program gives you](#what-the-program-gives-you)
2. [Start-up: console questions](#start-up-console-questions)
3. [Camera window and keys](#camera-window-and-keys)
4. [Preparing a measurement](#preparing-a-measurement)
5. [Experiment modes](#experiment-modes)
6. [Running an experiment](#running-an-experiment)
7. [Mode matrix](#mode-matrix)
8. [Real-time noise graph](#real-time-noise-graph)
9. [Results and files](#results-and-files)
10. [Typical workflow](#typical-workflow)
11. [Common problems](#common-problems)

## What the program gives you

In the end you get numbers that describe the camera and can be compared between cameras, capture modes and conditions:

- **Dark noise and dark non-uniformity** (dark mode): the noise floor of the camera, below which no signal can be seen, and the fixed pattern of the sensor without light.
- **Noise and non-uniformity under light** (light mode): how noise grows with light and how differently pixels respond to the same light.
- **Stability over time** (stability mode): whether the signal level drifts or the noise grows during long runs, for example as the camera warms up.
- **Signal-to-noise ratio for a blinking source** (modulator mode): how reliably the camera tells the light being on from off.
- **Comparison of camera modes** (mode matrix): which resolution, frame rate and format give the lowest noise.

All numbers are saved in a text report and in CSV tables that open in Excel.

## Start-up: console questions

After the start the program asks these questions in the console, in this order.

1. **Language.** `[1] English` or `[2] Русский`. Pressing Enter selects English. The choice sets the language of the console, the camera window and the reports.
2. **Camera backend** (Windows only): how the program talks to the camera.
   - `[1] Media Foundation`: the standard one.
   - `[2] DirectShow`: **recommended for experiments**. Only with it do the auto controls lock (key L) and the driver settings dialog (key O) work.
3. **Camera.** The program lists the cameras found and their default settings. Enter the camera number, or `q` to quit.
4. **Measuring camera modes.** The program goes through resolutions and frame rates and measures how many frames per second the camera *really* delivers. This takes about a minute. Keep normal lighting during the measurement: in the dark webcams lower the frame rate on their own.
   - The result is saved. On the next start the program offers to use the saved modes (`y`) or to measure again (`n`).
   - The measurement can be skipped (`n`). The standard list of modes is used then, and the camera may not support all of them.
5. **Modes table.** For each mode it shows the requested frame rate, the measured one and the one the driver reports. A frame rate is marked "not confirmed" when the real rate is more than 20% off the request. Such frame rates are not offered for selection.
6. **Change camera settings?** `y`: choose the mode, frame rate and pixel format. `n`: keep the camera defaults.
7. **Mode matrix.** The numbers of the modes to compare automatically (e.g. `1,3,5` or `all`), or `n` to skip. See [Mode matrix](#mode-matrix).

Then the camera window opens.

## Camera window and keys

The menu is on the left, information panels on the right, the experiment panel and notifications at the bottom. Text size follows the frame resolution. Press letter keys with the English keyboard layout.

### Experiments

| Key | Action |
|---|---|
| B | Start or stop an experiment |
| G | Next experiment mode: dark → light → stability → modulator |
| H | Next duration: 30 s, 1, 5, 10, 30 min |
| − / = | Lower or raise the noise threshold (0.1 DN step) |
| V | Open or close the noise graph window |
| U | Start or abort the mode matrix |
| L | Lock auto exposure, white balance and focus (or restore auto) |
| O | Open the camera driver settings dialog (DirectShow only) |

### Camera

| Key | Action |
|---|---|
| ← / → | Previous / next mode (resolution and format) |
| ↑ / ↓ | Next / previous frame rate |
| X | Screenshot (`screenshots` folder) |
| R | Start or stop video recording (`videos` folder) |
| ESC | Close the window and return to camera selection |

### Image filters

Filters only change the picture on screen. **They do not affect measurements**: noise is always computed on the raw camera frame.

| Key | Action |
|---|---|
| 1–5 | Toggle brightness/contrast, blur, edges, gray, sharpen |
| W / S, A / D | Brightness, contrast |
| E / Q | Blur size |
| J / K, N / M | Edge thresholds T1, T2 |
| T, P, F, C | Show/hide statistics, parameters, filter states, filter keys |
| TAB | Hide/show the menu |

### Experiment panel (bottom left)

- `Exp: dark [G] | 1m [H] | thr 2.0 DN [-/=]`: the selected mode, duration and threshold.
- `Noise 1.23 / 2.0 DN OK | Level 12.3`: temporal noise of the current frame, the threshold, the verdict and the average frame brightness. "OK" (green) means the noise is not above the threshold. "HIGH" (red) means it is above.
- During an experiment: `EXP dark 00:42 / 1m [B] stop`.

## Preparing a measurement

For reliable results:

1. **Lock the auto controls (key L)** before every experiment. Otherwise the camera changes exposure and gain on its own, and level and noise change because of the automation, not the sensor. The report warns when the lock was off.
   - On Windows, choose the DirectShow backend at start.
   - If L did not work, press O and clear the "Auto" boxes in the driver settings by hand.
   - On macOS the lock is not available: OpenCV's standard access to the camera (AVFoundation) does not give control over these settings.
2. **Fix the camera in place.** In every mode except the modulator, the camera and the scene must not move.
3. **Let the camera warm up** for 1–2 minutes after the start when stability matters.
4. **Avoid saturation** in light modes: noise in saturated pixels is underestimated. The report warns about saturated pixels.

## Experiment modes

The mode is selected with G. The mode determines the setup hint and how the report interprets the results.

### Dark

- **Setup:** cover the lens completely (with a cap or thick opaque cloth).
- **Measures:** camera noise without light (dark noise) and the fixed pattern of the sensor in the dark (DSNU).
- **Gives you:** the noise floor of the camera. A signal weaker than this cannot be seen.
- **Warnings:** an average level above 20 DN means the lens is not fully covered. More than 5% of pixels that never change means the camera clips black at zero and dark noise is underestimated.

### Light

- **Setup:** point the camera at an evenly lit surface or source (a matte screen, a sheet of paper under even light). No saturation.
- **Measures:** noise under light (mostly photon shot noise) and the difference in pixel sensitivity (PRNU).
- **Gives you:** how the camera behaves in working conditions and how evenly it renders the same light across the field.

### Stability

- **Setup:** fixed camera, static scene, constant light.
- **Measures:** change of level (drift) and noise over time, differences between zones of the frame.
- **Gives you:** whether long measurements can be trusted or the camera "wanders", e.g. as it warms up. Use 10–30 minutes for this mode.

### Modulator

- **Setup:** the camera does not move, the light is switched on and off periodically (by hand or by a modulator). The brightness difference between on and off must be at least 5 DN.
- **Measures:** the program splits frames into "light on" and "light off" by itself and computes the noise of each state, the signal (difference of levels), the signal-to-noise ratio (SNR), the number of switches and the period.
- **Gives you:** how confidently the camera tells the light being on from off. This is the key figure for detecting a modulated signal.
- The camera window shows the current state next to the mode: `[on]`, `[off]` or `[transition]`. Frames right after a switch are not used.

## Running an experiment

1. Select the mode (G), duration (H) and threshold (−/=).
2. Prepare the camera as the mode hint says and press L.
3. Press B. A timer appears at the bottom, and the console shows the folder of the results.
4. The experiment ends on its own when the time is up. B stops it early, and the results are saved then too.
5. A short summary appears in the notification and in the console. The full report is in `report.txt`.

While an experiment runs, the mode, duration and threshold cannot be changed. Changing the resolution or frame rate restarts the camera and stops the experiment.

### Choosing the threshold

The threshold does not change the measurements. It only marks frames and pixels whose noise is above the given value. Run a short dark measurement first and look at the average temporal noise, then set the threshold a little above it. The percentage of frames above the threshold then shows how often the camera departs from its normal level.

## Mode matrix

The matrix runs the same experiment in several camera modes one after another and puts the results into one table.

1. When the program asks about the matrix at start, enter the mode numbers from the table (e.g. `1,3,5` or `all`). Each mode is measured at all its confirmed frame rates.
2. In the camera window choose the experiment mode, duration and threshold, prepare the camera and press L.
3. Press U. The program switches the camera to the first mode, waits 3 seconds (warm-up), runs the experiment and moves on to the next mode.
4. U aborts the matrix. The modes already measured are kept.

The result is in `experiments/<date>_matrix_<mode>/`:

- `summary.txt`: a table for reading.
- `summary.csv`: the same table for Excel, with the folder of each mode's detailed report.

**Gives you:** the camera mode with the lowest noise or the best SNR for your conditions.

## Real-time noise graph

V opens a separate window with two graphs of the last 60 seconds:

- **Temporal noise** with the threshold line. The curve turns red when the noise is above the threshold.
- **Level** (average frame brightness).

While the window is open, the values are written to `noise_log/noise_<date>.csv`. The graph helps to set up the light and the threshold before an experiment and shows flicker or drift right away.

## Results and files

Every experiment is saved in its own folder: `experiments/<date>_<mode>_<resolution>@<fps>_<format>/`.

| File | Contents |
|---|---|
| `meta.txt` | Capture conditions. Written at the start, so it survives a crash |
| `report.txt` | The full report (see below) |
| `data.csv` | Values of every frame |
| `mean.png` | The average frame over the run |
| `noise_map.png` | Per-pixel noise map: the brighter, the noisier |
| `defects.png` | Defective pixels: blue never change, red are very noisy |

In the modulator mode the pictures have `_on` and `_off` suffixes: separately for light on and off.

### What is in report.txt

1. **Conditions:** mode, duration, threshold, camera, backend, resolution, frame rate, format, whether auto controls were locked, exposure/gain values.
2. **Over time:** level, drift and trend, temporal noise (average, min, max, spread), share of frames above the threshold.
3. **Modulation** (modulator only): switches, segment lengths, period, on/off levels and noise, signal, SNR.
4. **Across the field:** mean level, non-uniformity (DSNU or PRNU), row and column stripes (FPN), pixel noise, never-changing, noisy and saturated pixels, an 8×6 table of zones "level / noise".
5. **Warnings:** what in the conditions may have distorted the result.
6. **Definitions and the list of files.**

### data.csv

The table opens in Excel with a double click: `;` separator, decimal comma.

| Column | Meaning |
|---|---|
| `frame` | Frame number |
| `t_sec` | Time from the start, s |
| `mean` | Average level of the frame, DN |
| `drift` | Level minus the level of the first frame, DN |
| `spatial_std` | Spread of pixel values within the frame, DN |
| `temporal_noise` | Temporal noise of the frame, DN |
| `above_threshold` | 1 when the noise is above the threshold |
| `state` | Modulator: `on`, `off`, `transition`, `unknown` |

Columns of the matrix `summary.csv` and of `noise_log/*.csv` are listed in the [glossary](glossary.md#csv-columns).

### Other folders

| Folder | Contents |
|---|---|
| `screenshots/` | Screenshots (X) |
| `videos/` | Video recordings (R) |
| `noise_log/` | Data of the noise graph window (V) |
| `cache/` | Saved results of the camera mode measurement |
| `logs/` | Technical log of the program, useful for finding errors |

## Typical workflow

1. Start the program, choose the language, the DirectShow backend (on Windows) and the camera.
2. Measure the camera modes under normal light. Frame rates that are not confirmed are not actually delivered by the camera.
3. Choose a working mode or mark several modes for the matrix.
4. Open the noise graph (V), set up the light and the threshold.
5. Press L.
6. Run the experiments:
   - dark, 1–5 minutes: the base noise of the camera;
   - light, 1–5 minutes: noise and uniformity under light;
   - stability, 10–30 minutes: drift;
   - modulator: SNR for a blinking light.
7. To compare camera modes, run the matrix (U).
8. Compare the `report.txt` files or open the CSV files in Excel for charts.

## Common problems

**The modes table says "not confirmed".** The camera does not deliver that frame rate in that mode, even though the driver "accepts" it. This is typical for high resolutions without MJPG or for low light. Choose a confirmed frame rate or the MJPG format.

**"Auto controls not supported by … backend".** On Windows restart the program and choose DirectShow. On macOS the lock is not possible.

**"Fixed at current values (driver does not confirm)".** The program wrote the current exposure, white balance and focus values, but the driver does not report whether the automation is off. To be sure, check the driver settings (O).

**"Warning: requested format MJPG, camera uses YUY2".** The camera does not support the chosen format in this mode, so it uses another one. The actual format is recorded in the report.

**High dark level / many never-changing pixels.** The lens is not fully covered, or the camera clips the black level. The report warnings say which.

**Many late frames.** The computer cannot keep up with processing. Close the graph window, turn off the filters or choose a lower resolution.