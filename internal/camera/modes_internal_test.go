package camera

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFPSResultConfirmed(t *testing.T) {
	tests := []struct {
		name string
		r    FPSResult
		want bool
	}{
		{"exact", FPSResult{Requested: 30, Measured: 29.9}, true},
		{"within 20%", FPSResult{Requested: 30, Measured: 24.5}, true},
		{"driver ignores request", FPSResult{Requested: 60, Declared: 60, Measured: 30}, false},
		{"just over 20%", FPSResult{Requested: 24, Measured: 30}, false},
		{"no frames", FPSResult{Requested: 5}, false},
		{"size changed", FPSResult{Requested: 60, Measured: 60, Resized: true}, false},
	}
	for _, tt := range tests {
		if got := tt.r.Confirmed(); got != tt.want {
			t.Errorf("%s: Confirmed() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestConfirmedFPSAndResultLines(t *testing.T) {
	results := []FPSResult{
		{Requested: 5, Declared: 5, Measured: 5.1},
		{Requested: 15, Declared: 15, Measured: 15},
		{Requested: 30, Declared: 30, Measured: 29.9},
		{Requested: 60, Declared: 60, Measured: 30.1},
		{Requested: 90, Declared: 90, Measured: 30},
		{Requested: 120, Declared: 120, Measured: 29.8},
	}

	if got := confirmedFPS(results); !reflect.DeepEqual(got, []float64{5, 15, 30}) {
		t.Errorf("confirmedFPS = %v, want [5 15 30]", got)
	}

	lines := resultLines(results)
	if len(lines) != 4 {
		t.Fatalf("resultLines gave %d lines, want 3 confirmed + 1 merged: %q", len(lines), lines)
	}
	for _, want := range []string{"60-120", "30.0", "not confirmed"} {
		if !strings.Contains(lines[3], want) {
			t.Errorf("merged line %q does not contain %q", lines[3], want)
		}
	}
}

func TestUnconfirmedLine(t *testing.T) {
	line := unconfirmedLine([]FPSResult{
		{Requested: 5, Declared: 30, Measured: 29.6},
		{Requested: 10, Declared: 30, Measured: 30.4},
	})
	if !strings.Contains(line, "29.6-30.4") || strings.Contains(line, "30-30") {
		t.Errorf("unconfirmedLine = %q, want measured range 29.6-30.4 and a single driver value", line)
	}
}

func TestFormatLabel(t *testing.T) {
	tests := []struct {
		m    Mode
		want string
	}{
		{Mode{}, "auto"},
		{Mode{ActualFormat: "unknown"}, "auto"},
		{Mode{ActualFormat: "YUY2"}, "auto (YUY2)"},
		{Mode{Format: "MJPG", ActualFormat: "MJPG"}, "MJPG"},
		{Mode{Format: "MJPG", ActualFormat: "YUY2"}, "MJPG -> YUY2"},
		{Mode{Format: "MJPG", ActualFormat: "unknown"}, "MJPG (unverified)"},
	}
	for _, tt := range tests {
		if got := formatLabel(tt.m); got != tt.want {
			t.Errorf("formatLabel(%+v) = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestFindMode(t *testing.T) {
	modes := []Mode{
		{Width: 640, Height: 480, FPS: []float64{15, 30}},
		{Width: 1280, Height: 720, FPS: []float64{10, 30, 60}},
		{Width: 1280, Height: 720, Format: "MJPG", FPS: []float64{30, 60}},
	}

	tests := []struct {
		name          string
		w, h          int
		fps           float64
		format        string
		wantRes, want int
	}{
		{"exact", 1280, 720, 60, "", 1, 2},
		{"closest fps", 1280, 720, 25, "", 1, 1},
		{"format variant preferred", 1280, 720, 60, "MJPG", 2, 1},
		{"unknown format falls back to size", 1280, 720, 10, "NV12", 1, 0},
		{"unknown resolution falls back to first", 1024, 768, 30, "", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, fps := FindMode(modes, tt.w, tt.h, tt.fps, tt.format)
			if res != tt.wantRes || fps != tt.want {
				t.Errorf("FindMode = (%d, %d), want (%d, %d)", res, fps, tt.wantRes, tt.want)
			}
		})
	}

	if res, fps := FindMode(nil, 640, 480, 30, ""); res != 0 || fps != 0 {
		t.Errorf("FindMode(nil) = (%d, %d), want (0, 0)", res, fps)
	}
}

func TestSortModes(t *testing.T) {
	modes := []Mode{
		{Width: 1280, Height: 720, Format: "YUY2"},
		{Width: 1920, Height: 1080},
		{Width: 1280, Height: 720, Format: "MJPG"},
		{Width: 640, Height: 480},
		{Width: 1280, Height: 720},
	}
	sortModes(modes)

	var got []string
	for _, m := range modes {
		got = append(got, m.String())
	}
	want := []string{"640x480", "1280x720", "1280x720 MJPG", "1280x720 YUY2", "1920x1080"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortModes = %v, want %v", got, want)
	}
}

func TestModeCache(t *testing.T) {
	t.Chdir(t.TempDir())

	cam := Config{Name: "Camera 1", Width: 1280, Height: 720, FPS: 30, Format: "MJPG"}
	modes := []Mode{{Width: 640, Height: 480, ActualFormat: "YUY2", FPS: []float64{30},
		Results: []FPSResult{{Requested: 30, Declared: 30, Measured: 29.9}}}}

	if _, ok := loadModes(cam); ok {
		t.Fatal("loadModes found a cache before saving")
	}
	if err := saveModes(cam, modes); err != nil {
		t.Fatal(err)
	}

	c, ok := loadModes(cam)
	if !ok || !reflect.DeepEqual(c.Modes, modes) {
		t.Errorf("loadModes = %+v %v, want saved modes", c.Modes, ok)
	}

	other := cam
	other.Width = 640 // a different device at the same position
	if _, ok := loadModes(other); ok {
		t.Error("cache of another device was accepted")
	}

	if p := cachePath(Config{Name: "Camera 1"}); p != filepath.Join(CacheDir, "modes_Camera_1.json") {
		t.Errorf("cachePath = %q", p)
	}
	if p := cachePath(Config{Name: "Camera 1", API: APIDShow}); p != filepath.Join(CacheDir, "modes_Camera_1_DirectShow.json") {
		t.Errorf("cachePath with backend = %q", p)
	}
}

func TestFormatFPS(t *testing.T) {
	if got := FormatFPS([]float64{15, 29.97, 30}); got != "15, 29.97, 30" {
		t.Errorf("FormatFPS = %q", got)
	}
	if got := FormatFPS(nil); got != "none" {
		t.Errorf("FormatFPS(nil) = %q", got)
	}
}

func TestParseSelection(t *testing.T) {
	tests := []struct {
		input   string
		want    []int
		wantErr bool
	}{
		{"all", []int{0, 1, 2}, false},
		{"3,1", []int{2, 0}, false},
		{"2, 2", []int{1}, false},
		{"0", nil, true},
		{"4", nil, true},
		{"1,x", nil, true},
	}

	for _, tt := range tests {
		got, err := ParseSelection(tt.input, 3)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSelection(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseSelection(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFindCameraByNumber(t *testing.T) {
	cameras := []Config{{Index: 0, Name: "Camera 1"}, {Index: 2, Name: "Camera 2"}}

	if cam, ok := findCameraByNumber(cameras, 2); !ok || cam.Index != 2 {
		t.Errorf("number 2 = %+v %v, want device index 2", cam, ok)
	}
	for _, n := range []int{0, 3} {
		if _, ok := findCameraByNumber(cameras, n); ok {
			t.Errorf("number %d found, want not found", n)
		}
	}
}
