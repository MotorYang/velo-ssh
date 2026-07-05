package term

import (
	"os"
	"runtime"
	"strings"

	"github.com/motoryang/velo-ssh/internal/config"
)

func ShouldUseASCII(setting string) bool {
	switch setting {
	case config.ASCIIFallbackAlways:
		return true
	case config.ASCIIFallbackDisabled:
		return false
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if runtime.GOOS == "linux" && (term == "linux" || term == "dumb" || term == "") {
		return true
	}
	return false
}
