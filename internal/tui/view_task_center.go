package tui

import (
	"fmt"
	"sort"

	"github.com/motoryang/velo-ssh/internal/term"
	"github.com/motoryang/velo-ssh/internal/transfer"
)

func (m Model) viewTaskCenter() string {
	width := m.contentWidth()
	inner := width - 2
	body := []string{}
	tasks := m.taskSnapshots()
	if len(tasks) == 0 {
		body = append(body, "No transfer tasks.")
	}
	cursor := clampCursor(m.taskCursor, len(tasks))
	for i, task := range tasks {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		percent := 0
		if task.BytesTotal > 0 {
			percent = int(task.BytesDone * 100 / task.BytesTotal)
		}
		line := fmt.Sprintf("%s%-8s %-9s %3d%% %8s/%-8s %s -> %s",
			prefix,
			task.Direction,
			task.Status,
			percent,
			humanBytes(task.BytesDone),
			humanBytes(task.BytesTotal),
			task.SourcePath,
			task.TargetPath,
		)
		if task.Error != "" {
			line += " " + task.Error
		}
		line = term.Truncate(line, inner)
		if i == cursor {
			line = m.styles.selected.Render(line)
		}
		body = append(body, line)
	}
	return borderedBlock("Task Center", width, body)
}

func (m Model) taskSnapshots() []*transfer.Task {
	tasks := m.tasks.Tasks()
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].StartedAt.Equal(tasks[j].StartedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].StartedAt.Before(tasks[j].StartedAt)
	})
	return tasks
}

func humanBytes(size int64) string {
	if size <= 0 {
		return "-"
	}
	value := float64(size)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d%s", size, units[unit])
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}
