package node

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int // >0, <0, or 0
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.4", "1.2.3", 1},
		{"1.2.3", "1.2.4", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.10.0", "1.9.0", 1},
		{"v1.2.3", "1.2.3", 0},
		{"openclaw 1.2.3", "1.2.3", 0},
		{"openclaw 1.3.0", "v1.2.9", 1},
		{"zeroclaw 0.5.0", "0.4.9", 1},
		{"1.0.0", "2.0.0", -1},
		{"", "", 0},
		{"1.0.0", "", 1},
		{"", "1.0.0", -1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		switch {
		case tt.want > 0 && got <= 0:
			t.Errorf("CompareVersions(%q, %q) = %d, want >0", tt.a, tt.b, got)
		case tt.want < 0 && got >= 0:
			t.Errorf("CompareVersions(%q, %q) = %d, want <0", tt.a, tt.b, got)
		case tt.want == 0 && got != 0:
			t.Errorf("CompareVersions(%q, %q) = %d, want 0", tt.a, tt.b, got)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"openclaw 1.2.3", [3]int{1, 2, 3}},
		{"zeroclaw 0.5.1", [3]int{0, 5, 1}},
		{"1.0", [3]int{1, 0, 0}},
		{"3", [3]int{3, 0, 0}},
		{"", [3]int{0, 0, 0}},
		{"v2.10.3 extra", [3]int{2, 10, 3}},
	}

	for _, tt := range tests {
		got := parseVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
