package transfer

import (
	"sync"
	"time"
)

type Direction string

const (
	Upload   Direction = "upload"
	Download Direction = "download"
)

type Status string

const (
	TaskQueued    Status = "queued"
	TaskRunning   Status = "running"
	TaskPaused    Status = "paused"
	TaskSucceeded Status = "succeeded"
	TaskFailed    Status = "failed"
	TaskCanceled  Status = "canceled"
)

type Task struct {
	mu         sync.Mutex
	ID         string
	ServerID   string
	Direction  Direction
	SourcePath string
	TargetPath string
	BytesTotal int64
	BytesDone  int64
	Status     Status
	Error      string
	StartedAt  time.Time
	UpdatedAt  time.Time
	cancel     chan struct{}
	paused     bool
	pausedFrom Status
}

func NewTask(id string, direction Direction, source, target string) *Task {
	now := time.Now()
	return &Task{
		ID:         id,
		Direction:  direction,
		SourcePath: source,
		TargetPath: target,
		Status:     TaskQueued,
		StartedAt:  now,
		UpdatedAt:  now,
		cancel:     make(chan struct{}),
	}
}

func (t *Task) Snapshot() Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	return Task{
		ID:         t.ID,
		ServerID:   t.ServerID,
		Direction:  t.Direction,
		SourcePath: t.SourcePath,
		TargetPath: t.TargetPath,
		BytesTotal: t.BytesTotal,
		BytesDone:  t.BytesDone,
		Status:     t.Status,
		Error:      t.Error,
		StartedAt:  t.StartedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func (t *Task) SetProgress(done, total int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.BytesDone = done
	t.BytesTotal = total
	t.UpdatedAt = time.Now()
}

func (t *Task) Pause() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Status == TaskQueued || t.Status == TaskRunning {
		t.pausedFrom = t.Status
		t.paused = true
		t.Status = TaskPaused
		t.UpdatedAt = time.Now()
	}
}

func (t *Task) Resume() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Status == TaskPaused {
		t.paused = false
		if t.pausedFrom == TaskRunning {
			t.Status = TaskRunning
			t.Error = "resume continued sequential transfer"
		} else {
			t.Status = TaskQueued
			t.Error = "resume restarted or continued sequential transfer"
		}
		t.pausedFrom = ""
		t.UpdatedAt = time.Now()
	}
}

func (t *Task) WaitIfPaused(canceled <-chan struct{}) bool {
	for {
		t.mu.Lock()
		paused := t.paused
		t.mu.Unlock()
		if !paused {
			return true
		}
		select {
		case <-canceled:
			return false
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (t *Task) IsCanceled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status == TaskCanceled
}

func (t *Task) IsFinished() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status == TaskSucceeded || t.Status == TaskFailed || t.Status == TaskCanceled
}

func (t *Task) Cancel() {
	t.mu.Lock()
	cancel := t.cancel
	t.Status = TaskCanceled
	t.UpdatedAt = time.Now()
	t.mu.Unlock()
	select {
	case <-cancel:
	default:
		close(cancel)
	}
}

func (t *Task) mark(status Status, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
	if err != nil {
		t.Error = err.Error()
	}
	t.UpdatedAt = time.Now()
}
