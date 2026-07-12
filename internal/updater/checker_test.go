package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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

func TestSelectAssetMatchesPlatformArchive(t *testing.T) {
	rel := Release{
		Version: "v1.0.0.26070602",
		Assets: []Asset{
			{Name: "velossh-linux-amd64.tar.gz", DownloadURL: "https://example.com/linux"},
			{Name: "velossh-windows-arm64.zip", DownloadURL: "https://example.com/windows"},
		},
	}
	asset, err := SelectAsset(rel, "windows", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if asset.DownloadURL != "https://example.com/windows" {
		t.Fatalf("download URL = %q", asset.DownloadURL)
	}
}

func TestSelectAssetReportsMissingPlatform(t *testing.T) {
	rel := Release{Version: "v1.0.0.26070602"}
	if _, err := SelectAsset(rel, "linux", "arm64"); err == nil {
		t.Fatal("expected missing asset error")
	}
}

func TestEnsureInstallTargetWritableReportsRecoveryCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions test")
	}
	dir := filepath.Join(t.TempDir(), "bin")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o700)
	err := ensureInstallTargetWritable(filepath.Join(dir, "vssh"), "v1.2.3")
	if err == nil {
		t.Skip("install target remained writable, likely running with elevated permissions")
	}
	got := err.Error()
	for _, want := range []string{
		"not writable",
		dir,
		"VERSION=v1.2.3",
		"scripts/install.sh",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error missing %q: %s", want, got)
		}
	}
}
