package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{a: "v1.0.0.26070602", b: "v1.0.0.26070601", want: 1},
		{a: "v1.0.1.1", b: "v1.0.0.99999999", want: 1},
		{a: "v1.0.0.26070601", b: "v1.0.0.26070601", want: 0},
		{a: "v1.0.0.26070600", b: "v1.0.0.26070601", want: -1},
	}
	for _, tt := range tests {
		if got := CompareVersions(tt.a, tt.b); got != tt.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
