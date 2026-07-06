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
