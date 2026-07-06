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

func TestMatcherUsesDefaultIgnoreRules(t *testing.T) {
	m := New(nil)
	for _, rel := range []string{".DS_Store", "nested/Thumbs.db", ".git/config"} {
		if !m.Ignored(rel, false) {
			t.Fatalf("expected default ignore for %q", rel)
		}
	}
}

func TestUserRulesCanNegateDefaultIgnore(t *testing.T) {
	m := New([]string{"!.git/"})
	if m.Ignored(".git/config", false) {
		t.Fatal("expected user negation to include .git/config")
	}
}
