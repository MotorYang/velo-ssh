package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
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
	if rel.Version == "" {
		return Release{}, false, fmt.Errorf("check latest release: missing tag_name")
	}
	return rel, CompareVersions(rel.Version, current) > 0, nil
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
