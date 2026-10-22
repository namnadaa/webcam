package camera

import (
	"reflect"
	"testing"
)

func TestAddMode(t *testing.T) {
	var modes []Mode
	modes = addMode(modes, 1280, 720, 30)
	modes = addMode(modes, 640, 480, 30)
	modes = addMode(modes, 640, 480, 15)
	modes = addMode(modes, 640, 480, 30.001) // rounds to an existing value
	modes = addMode(modes, 1280, 720, 29.97)
	modes = addMode(modes, 800, 600, 0) // unknown fps keeps the resolution only
	modes = addMode(modes, 0, 0, 30)    // invalid size is ignored

	want := []Mode{
		{Width: 640, Height: 480, FPS: []float64{15, 30}},
		{Width: 800, Height: 600},
		{Width: 1280, Height: 720, FPS: []float64{29.97, 30}},
	}

	if !reflect.DeepEqual(modes, want) {
		t.Errorf("addMode result = %+v, want %+v", modes, want)
	}
}

func TestFindMode(t *testing.T) {
	modes := []Mode{
		{Width: 640, Height: 480, FPS: []float64{15, 30}},
		{Width: 1280, Height: 720, FPS: []float64{10, 30, 60}},
	}

	tests := []struct {
		name          string
		w, h          int
		fps           float64
		wantRes, want int
	}{
		{"exact", 1280, 720, 60, 1, 2},
		{"closest fps", 1280, 720, 25, 1, 1},
		{"unknown resolution falls back to first", 1024, 768, 30, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, fps := FindMode(modes, tt.w, tt.h, tt.fps)
			if res != tt.wantRes || fps != tt.want {
				t.Errorf("FindMode = (%d, %d), want (%d, %d)", res, fps, tt.wantRes, tt.want)
			}
		})
	}

	if res, fps := FindMode(nil, 640, 480, 30); res != 0 || fps != 0 {
		t.Errorf("FindMode(nil) = (%d, %d), want (0, 0)", res, fps)
	}
}

func TestFormatFPS(t *testing.T) {
	if got := FormatFPS([]float64{15, 29.97, 30}); got != "15, 29.97, 30" {
		t.Errorf("FormatFPS = %q", got)
	}
	if got := FormatFPS(nil); got != "unknown" {
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
