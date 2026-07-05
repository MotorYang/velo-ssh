package term

import (
	"testing"

	"github.com/motoryang/velo-ssh/internal/config"
)

func TestShouldUseASCIIExplicitSettings(t *testing.T) {
	if !ShouldUseASCII(config.ASCIIFallbackAlways) {
		t.Fatal("always should force ASCII")
	}
	if ShouldUseASCII(config.ASCIIFallbackDisabled) {
		t.Fatal("disabled should not force ASCII")
	}
}

func TestWidthCJK(t *testing.T) {
	if Width("中") != 2 {
		t.Fatalf("CJK width = %d, want 2", Width("中"))
	}
}
