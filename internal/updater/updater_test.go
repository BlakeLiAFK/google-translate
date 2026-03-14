package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v0.9.0", 1},
		{"v1.0.0", "v1.0.0", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.1.0", "v1.0.9", 1},
		{"v0.1.0", "dev", 1},
		{"v1.0.0-beta", "v0.9.9", 1},
		{"v1.0.0", "v1.0.0-beta", 0}, // beta 后缀被忽略
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.0.0", [3]int{1, 0, 0}},
		{"dev", [3]int{0, 0, 0}},
		{"v2.1.0-beta", [3]int{2, 1, 0}},
	}
	for _, tt := range tests {
		got := parseVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
