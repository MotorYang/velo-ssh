package term

import "github.com/mattn/go-runewidth"

func Width(s string) int {
	return runewidth.StringWidth(s)
}

func Truncate(s string, width int) string {
	return runewidth.Truncate(s, width, "…")
}
