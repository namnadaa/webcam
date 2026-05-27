package helpers

import "time"

// onOff converts a boolean value into an ON/OFF status string.
func OnOff(v bool) string {
	if v {
		return "ON "
	}
	return "OFF"
}

// UpdateFPS calculates current FPS value.
func UpdateFPS(frameCount *int, lastFPS *time.Time, fps *float64) {
	*frameCount++

	elapsed := time.Since(*lastFPS)

	if elapsed >= time.Second {
		*fps = float64(*frameCount) / elapsed.Seconds()

		*frameCount = 0
		*lastFPS = time.Now()
	}
}
