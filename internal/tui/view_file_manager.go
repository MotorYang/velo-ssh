package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/term"
)

const minFilePaneWidth = 54

func (m Model) viewFileManager() string {
	var b strings.Builder
	fmt.Fprintln(&b, m.styles.title.Render("File Manager"))
	leftMarker := " "
	rightMarker := " "
	if m.activePane == 0 {
		leftMarker = ">"
	} else {
		rightMarker = ">"
	}
	split := m.config.Settings.DefaultViewMode == config.ViewSplit
	paneWidth := m.filePaneWidth(split)
	if !split {
		marker := ">"
		if m.activePane != 1 {
			marker = " "
		}
		fmt.Fprintf(&b, "%s REMOTE: %s\n", marker, m.remoteDir)
		if m.ssh == nil {
			fmt.Fprintln(&b, "  Remote pane requires an active SSH/SFTP connection.")
		}
		fmt.Fprintln(&b)
		start, end := visibleFileRange(len(m.remoteFiles), m.remoteCursor, m.fileViewportRows())
		fmt.Fprintf(&b, "REMOTE rows %d-%d/%d\n", displayStart(start, end), end, len(m.remoteFiles))
		fmt.Fprintln(&b, strings.Repeat("-", paneWidth))
		for i := start; i < end; i++ {
			item := m.remoteFiles[i]
			fmt.Fprintln(&b, padVisual(m.renderFileRow(i, item, m.activePane == 1, m.remoteCursor, paneWidth), paneWidth))
		}
		if m.renaming {
			fmt.Fprintf(&b, "\nRename: %s\n", m.renameInput.View())
		}
		if m.creatingDir {
			fmt.Fprintf(&b, "\nNew directory: %s\n", m.mkdirInput.View())
		}
		return b.String()
	}
	fmt.Fprintf(&b, "%s LOCAL: %s\n", leftMarker, m.localDir)
	fmt.Fprintf(&b, "%s REMOTE: %s\n", rightMarker, m.remoteDir)
	if m.ssh == nil {
		fmt.Fprintln(&b, "  Remote pane requires an active SSH/SFTP connection.")
	}
	fmt.Fprintln(&b)
	rows := m.fileViewportRows()
	localStart, localEnd := visibleFileRange(len(m.localFiles), m.localCursor, rows)
	remoteStart, remoteEnd := visibleFileRange(len(m.remoteFiles), m.remoteCursor, rows)
	fmt.Fprintf(&b, "%s | %s\n",
		padVisual(fmt.Sprintf("LOCAL rows %d-%d/%d", displayStart(localStart, localEnd), localEnd, len(m.localFiles)), paneWidth),
		padVisual(fmt.Sprintf("REMOTE rows %d-%d/%d", displayStart(remoteStart, remoteEnd), remoteEnd, len(m.remoteFiles)), paneWidth),
	)
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", paneWidth*2+3))
	for row := 0; row < rows; row++ {
		left := ""
		right := ""
		localIndex := localStart + row
		remoteIndex := remoteStart + row
		if localIndex < localEnd && localIndex < len(m.localFiles) {
			left = m.renderFileRow(localIndex, m.localFiles[localIndex], m.activePane == 0, m.localCursor, paneWidth)
		}
		if remoteIndex < remoteEnd && remoteIndex < len(m.remoteFiles) {
			right = m.renderFileRow(remoteIndex, m.remoteFiles[remoteIndex], m.activePane == 1, m.remoteCursor, paneWidth)
		}
		fmt.Fprintf(&b, "%s | %s\n", padVisual(left, paneWidth), padVisual(right, paneWidth))
	}
	fmt.Fprintln(&b)
	if m.renaming {
		fmt.Fprintf(&b, "Rename: %s\n", m.renameInput.View())
	}
	if m.creatingDir {
		fmt.Fprintf(&b, "New directory: %s\n", m.mkdirInput.View())
	}
	return b.String()
}

func (m Model) renderFileRow(index int, item fileItem, focused bool, cursor int, paneWidth int) string {
	prefix := "  "
	if focused && index == cursor {
		prefix = "> "
	}
	check := "[ ]"
	if item.Selected {
		check = "[*]"
	}
	mode := formatMode(item)
	size := humanSize(item)
	nameWidth := paneWidth - 26
	if m.showFileTime {
		nameWidth -= 17
	}
	if nameWidth < 8 {
		nameWidth = 8
	}
	name := term.Truncate(item.Name, nameWidth)
	line := fmt.Sprintf("%s%s %s %s %8s", prefix, check, mode, padRightVisual(name, nameWidth), size)
	if m.showFileTime {
		line = fmt.Sprintf("%s %16s", line, formatModTime(item))
	}
	if focused && index == cursor {
		return m.styles.selected.Render(line)
	}
	return line
}

func (m Model) filePaneWidth(split bool) int {
	if m.width <= 0 {
		if split {
			return 80
		}
		return 120
	}
	if !split {
		width := m.width
		if width < minFilePaneWidth {
			return minFilePaneWidth
		}
		return width
	}
	width := (m.width - 3) / 2
	if width < minFilePaneWidth {
		return minFilePaneWidth
	}
	return width
}

func (m Model) fileViewportRows() int {
	if m.height <= 0 {
		return 20
	}
	rows := m.height - 10
	if m.renaming || m.creatingDir {
		rows -= 2
	}
	if m.err != "" {
		rows--
	}
	if m.status != "" {
		rows--
	}
	if rows < 5 {
		return 5
	}
	return rows
}

func visibleFileRange(total, cursor, rows int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if rows <= 0 || rows > total {
		rows = total
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	start := cursor - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > total {
		start = total - rows
	}
	return start, start + rows
}

func displayStart(start, end int) int {
	if end == 0 {
		return 0
	}
	return start + 1
}

func padVisual(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func padRightVisual(s string, width int) string {
	w := term.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func formatMode(item fileItem) string {
	mode := item.Mode
	if mode == 0 {
		if item.IsDir {
			mode = os.FileMode(0o755) | os.ModeDir
		} else {
			mode = 0o644
		}
	}
	return mode.String()
}

func humanSize(item fileItem) string {
	if item.IsDir {
		return "-"
	}
	size := float64(item.Size)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d%s", item.Size, units[unit])
	}
	return fmt.Sprintf("%.1f%s", size, units[unit])
}

func formatModTime(item fileItem) string {
	if item.ModTime.IsZero() {
		return "-"
	}
	return item.ModTime.Format("2006-01-02 15:04")
}
