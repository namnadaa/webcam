package helpers

import "testing"

func TestCSVFloat(t *testing.T) {
	tests := []struct {
		v    float64
		prec int
		want string
	}{
		{1.25, 2, "1,25"},
		{-0.5, 3, "-0,500"},
		{30, 0, "30"},
		{12.3456, 2, "12,35"},
	}
	for _, tt := range tests {
		if got := CSVFloat(tt.v, tt.prec); got != tt.want {
			t.Errorf("CSVFloat(%v, %d) = %q, want %q", tt.v, tt.prec, got, tt.want)
		}
	}
}

func TestCSVRow(t *testing.T) {
	if got := CSVRow("a", "1,5", ""); got != "a;1,5;\n" {
		t.Errorf("CSVRow = %q", got)
	}
}
