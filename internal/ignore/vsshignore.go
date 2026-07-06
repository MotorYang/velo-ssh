package ignore

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type Matcher struct {
	patterns []pattern
}

type pattern struct {
	raw     string
	negate  bool
	dirOnly bool
}

func LoadFile(filePath string) (Matcher, error) {
	patterns := defaultPatterns()
	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return Matcher{patterns: patterns}, nil
	}
	if err != nil {
		return Matcher{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if pat, ok := parseLine(scanner.Text()); ok {
			patterns = append(patterns, pat)
		}
	}
	if err := scanner.Err(); err != nil {
		return Matcher{}, err
	}
	return Matcher{patterns: patterns}, nil
}

func New(lines []string) Matcher {
	patterns := defaultPatterns()
	for _, line := range lines {
		if pat, ok := parseLine(line); ok {
			patterns = append(patterns, pat)
		}
	}
	return Matcher{patterns: patterns}
}

func defaultPatterns() []pattern {
	defaults := []string{".DS_Store", "Thumbs.db", ".git/"}
	patterns := make([]pattern, 0, len(defaults))
	for _, line := range defaults {
		if pat, ok := parseLine(line); ok {
			patterns = append(patterns, pat)
		}
	}
	return patterns
}

func (m Matcher) Empty() bool {
	return len(m.patterns) == 0
}

func (m Matcher) Ignored(rel string, isDir bool) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	rel = strings.TrimPrefix(rel, "./")
	if rel == "." || rel == "" {
		return false
	}
	ignored := false
	for _, pat := range m.patterns {
		if pat.matches(rel, isDir) {
			ignored = !pat.negate
		}
	}
	return ignored
}

func parseLine(line string) (pattern, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return pattern{}, false
	}
	pat := pattern{}
	if strings.HasPrefix(line, "\\#") || strings.HasPrefix(line, "\\!") {
		line = line[1:]
	} else if strings.HasPrefix(line, "!") {
		pat.negate = true
		line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
	}
	line = filepath.ToSlash(line)
	line = strings.TrimPrefix(line, "./")
	if strings.HasSuffix(line, "/") {
		pat.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}
	line = strings.Trim(line, "/")
	if line == "" {
		return pattern{}, false
	}
	pat.raw = line
	return pat, true
}

func (p pattern) matches(rel string, isDir bool) bool {
	if p.dirOnly {
		return rel == p.raw || strings.HasPrefix(rel, p.raw+"/")
	}
	if strings.Contains(p.raw, "/") {
		return globMatch(p.raw, rel)
	}
	if globMatch(p.raw, path.Base(rel)) {
		return true
	}
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		if globMatch(p.raw, part) {
			return true
		}
	}
	return false
}

func globMatch(pattern, value string) bool {
	re, err := regexp.Compile("^" + globToRegex(pattern) + "$")
	if err != nil {
		return pattern == value
	}
	return re.MatchString(value)
}

func globToRegex(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if ch == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString(`[^/]*`)
			}
			continue
		}
		if ch == '?' {
			b.WriteString(`[^/]`)
			continue
		}
		b.WriteString(regexp.QuoteMeta(string(ch)))
	}
	return b.String()
}
