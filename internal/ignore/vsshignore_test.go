package ignore

import "testing"

func TestMatcherIgnoresGlobDirectoryAndNegation(t *testing.T) {
	m := New([]string{
		"# comment",
		"*.log",
		"dist/",
		"tmp/**",
		"!keep.log",
	})
	tests := []struct {
		rel   string
		isDir bool
		want  bool
	}{
		{rel: "app.log", want: true},
		{rel: "nested/app.log", want: true},
		{rel: "keep.log", want: false},
		{rel: "dist", isDir: true, want: true},
		{rel: "dist/app.bin", want: true},
		{rel: "tmp/cache/file.txt", want: true},
		{rel: "src/main.go", want: false},
	}
	for _, tt := range tests {
		if got := m.Ignored(tt.rel, tt.isDir); got != tt.want {
			t.Fatalf("Ignored(%q, %v) = %v, want %v", tt.rel, tt.isDir, got, tt.want)
		}
	}
}
