package camera

// Resolution describes camera frame size.
type Resolution struct {
	Width  int
	Height int
}

// Resolutions contains predefined camera resolutions.
var Resolutions = []Resolution{
	{320, 240},   // QVGA
	{640, 480},   // VGA
	{800, 600},   // SVGA
	{1024, 768},  // XGA
	{1280, 720},  // HD
	{1600, 900},  // HD+
	{1920, 1080}, // Full HD
	{2560, 1440}, // 2K
	{3840, 2160}, // 4K
}

// FPSPresets contains predefined camera FPS values.
var FPSPresets = []float64{
	5,
	10,
	15,
	24,
	30,
	60,
	90,
	120,
	144,
	240,
}
