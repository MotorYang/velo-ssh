package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const LatestReleaseURL = "https://api.github.com/repos/MotorYang/velo-ssh/releases/latest"

type Release struct {
	Version string
	Name    string
	URL     string
	Body    string
	Assets  []Asset
}

type Asset struct {
	Name        string
	DownloadURL string
}

type Progress struct {
	Stage      string
	Downloaded int64
	Total      int64
}

func OpenURL(url string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("open update page: empty URL")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func CheckLatest(ctx context.Context, current string) (Release, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestReleaseURL, nil)
	if err != nil {
		return Release{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "VeloSSH/"+current)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Release{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Release{}, false, fmt.Errorf("check latest release: GitHub returned %s", resp.Status)
	}
	var payload latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Release{}, false, err
	}
	rel := Release{
		Version: strings.TrimSpace(payload.TagName),
		Name:    strings.TrimSpace(payload.Name),
		URL:     strings.TrimSpace(payload.HTMLURL),
		Body:    payload.Body,
	}
	for _, asset := range payload.Assets {
		name := strings.TrimSpace(asset.Name)
		downloadURL := strings.TrimSpace(asset.BrowserDownloadURL)
		if name != "" && downloadURL != "" {
			rel.Assets = append(rel.Assets, Asset{Name: name, DownloadURL: downloadURL})
		}
	}
	if rel.Version == "" {
		return Release{}, false, fmt.Errorf("check latest release: missing tag_name")
	}
	return rel, CompareVersions(rel.Version, current) > 0, nil
}

func InstallLatest(ctx context.Context, rel Release) error {
	return InstallLatestWithProgress(ctx, rel, nil)
}

func InstallLatestWithProgress(ctx context.Context, rel Release, progress func(Progress)) error {
	report := func(stage string, downloaded, total int64) {
		if progress != nil {
			progress(Progress{Stage: stage, Downloaded: downloaded, Total: total})
		}
	}
	report("selecting", 0, 0)
	asset, err := SelectAsset(rel, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	report("downloading", 0, 0)
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("install update: locate current executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("install update: resolve current executable: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := ensureInstallTargetWritable(exe, rel.Version); err != nil {
			return err
		}
	}
	tmpDir, err := os.MkdirTemp("", "velossh-update-*")
	if err != nil {
		return fmt.Errorf("install update: create temporary directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(tmpDir)
		}
	}()
	archivePath := filepath.Join(tmpDir, asset.Name)
	if err := downloadFile(ctx, asset.DownloadURL, archivePath, func(done, total int64) {
		report("downloading", done, total)
	}); err != nil {
		return err
	}
	report("extracting", 0, 0)
	binaryPath, err := extractAsset(archivePath, tmpDir, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	report("installing", 0, 0)
	if runtime.GOOS == "windows" {
		cleanup = false
		if err := scheduleWindowsReplace(exe, binaryPath, tmpDir); err != nil {
			cleanup = true
			return err
		}
		report("scheduled", 0, 0)
		return nil
	}
	if err := replaceExecutable(exe, binaryPath); err != nil {
		return err
	}
	report("installed", 0, 0)
	return nil
}

func SelectAsset(rel Release, goos, goarch string) (Asset, error) {
	wantBase := fmt.Sprintf("velossh-%s-%s", goos, goarch)
	wantSuffix := ".tar.gz"
	if goos == "windows" {
		wantSuffix = ".zip"
	}
	want := wantBase + wantSuffix
	for _, asset := range rel.Assets {
		if asset.Name == want {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("install update %s: no release asset for %s/%s; expected %s", rel.Version, goos, goarch, want)
}

func downloadFile(ctx context.Context, url, dst string, progress func(done, total int64)) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("install update: create download request: %w", err)
	}
	req.Header.Set("User-Agent", "VeloSSH updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("install update: download release asset: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("install update: download release asset: server returned %s", resp.Status)
	}
	total := resp.ContentLength
	if progress != nil {
		progress(0, total)
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("install update: create archive: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(&progressWriter{writer: out, total: total, progress: progress}, resp.Body); err != nil {
		return fmt.Errorf("install update: write archive: %w", err)
	}
	return nil
}

type progressWriter struct {
	writer   io.Writer
	done     int64
	total    int64
	progress func(done, total int64)
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.done += int64(n)
	if w.progress != nil {
		w.progress(w.done, w.total)
	}
	return n, err
}

func extractAsset(archivePath, tmpDir, goos, goarch string) (string, error) {
	name := "velossh-" + goos + "-" + goarch
	if goos == "windows" {
		name += ".exe"
		return extractZipFile(archivePath, tmpDir, name)
	}
	return extractTarGzFile(archivePath, tmpDir, name)
}

func extractZipFile(archivePath, tmpDir, targetName string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("install update: open zip asset: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) != targetName {
			continue
		}
		src, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("install update: open binary in zip: %w", err)
		}
		defer src.Close()
		dst := filepath.Join(tmpDir, targetName)
		if err := writeExtractedFile(dst, src, f.FileInfo().Mode()); err != nil {
			return "", err
		}
		return dst, nil
	}
	return "", fmt.Errorf("install update: binary %s not found in zip asset", targetName)
}

func extractTarGzFile(archivePath, tmpDir, targetName string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("install update: open tar.gz asset: %w", err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("install update: read tar.gz asset: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("install update: read tar.gz entry: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != targetName {
			continue
		}
		dst := filepath.Join(tmpDir, targetName)
		if err := writeExtractedFile(dst, tr, os.FileMode(header.Mode)); err != nil {
			return "", err
		}
		return dst, nil
	}
	return "", fmt.Errorf("install update: binary %s not found in tar.gz asset", targetName)
}

func writeExtractedFile(dst string, src io.Reader, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o755
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode|0o700)
	if err != nil {
		return fmt.Errorf("install update: create extracted binary: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("install update: write extracted binary: %w", err)
	}
	return nil
}

func replaceExecutable(current, next string) error {
	info, err := os.Stat(current)
	if err != nil {
		return fmt.Errorf("install update: stat current executable: %w", err)
	}
	if err := os.Chmod(next, info.Mode()|0o700); err != nil {
		return fmt.Errorf("install update: make new executable runnable: %w", err)
	}
	backup := current + ".vssh-old"
	_ = os.Remove(backup)
	if err := os.Rename(current, backup); err != nil {
		return fmt.Errorf("install update: move current executable aside: %w", err)
	}
	if err := os.Rename(next, current); err != nil {
		_ = os.Rename(backup, current)
		return fmt.Errorf("install update: install new executable: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

func ensureInstallTargetWritable(current, version string) error {
	dir := filepath.Dir(current)
	probe, err := os.CreateTemp(dir, ".vssh-update-probe-*")
	if err == nil {
		name := probe.Name()
		_ = probe.Close()
		_ = os.Remove(name)
		return nil
	}
	if os.IsPermission(err) {
		return fmt.Errorf("install update: %s is not writable by the current user; update from a terminal with: VERSION=%s sh -c \"$(curl -fsSL https://raw.githubusercontent.com/motoryang/velo-ssh/main/scripts/install.sh)\"", dir, version)
	}
	return fmt.Errorf("install update: check install target permissions: %w", err)
}

func scheduleWindowsReplace(current, next, tmpDir string) error {
	script := filepath.Join(tmpDir, "velossh-update.cmd")
	content := fmt.Sprintf("@echo off\r\nping 127.0.0.1 -n 3 > nul\r\ncopy /Y %q %q > nul\r\n", next, current)
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		return fmt.Errorf("install update: create Windows replacement script: %w", err)
	}
	cmd := exec.Command("cmd", "/C", "start", "", "/MIN", script)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("install update: start Windows replacement script: %w", err)
	}
	return nil
}

func CompareVersions(a, b string) int {
	aa := versionNumbers(a)
	bb := versionNumbers(b)
	max := len(aa)
	if len(bb) > max {
		max = len(bb)
	}
	for i := 0; i < max; i++ {
		var av, bv int
		if i < len(aa) {
			av = aa[i]
		}
		if i < len(bb) {
			bv = bb[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func versionNumbers(v string) []int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(v, -1)
	out := make([]int, 0, len(matches))
	for _, match := range matches {
		n, err := strconv.Atoi(match)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}
