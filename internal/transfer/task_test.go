package transfer

import (
	"testing"
	"time"
)

func TestTaskCancel(t *testing.T) {
	task := NewTask("1", Upload, "a", "b")
	task.Cancel()
	if task.Status != TaskCanceled {
		t.Fatalf("status = %s, want %s", task.Status, TaskCanceled)
	}
	select {
	case <-task.cancel:
	default:
		t.Fatal("cancel channel was not closed")
	}
}

func TestTaskPauseResumeQueued(t *testing.T) {
	task := NewTask("1", Upload, "a", "b")
	task.Pause()
	if got := task.Snapshot().Status; got != TaskPaused {
		t.Fatalf("paused status = %s, want %s", got, TaskPaused)
	}
	task.Resume()
	snapshot := task.Snapshot()
	if snapshot.Status != TaskQueued {
		t.Fatalf("resumed status = %s, want %s", snapshot.Status, TaskQueued)
	}
	if snapshot.Error == "" {
		t.Fatal("resume should expose best-effort resume status")
	}
}

func TestTaskPauseResumeRunningContinues(t *testing.T) {
	task := NewTask("1", Upload, "a", "b")
	task.mark(TaskRunning, nil)
	task.Pause()
	if got := task.Snapshot().Status; got != TaskPaused {
		t.Fatalf("paused status = %s, want %s", got, TaskPaused)
	}
	task.Resume()
	snapshot := task.Snapshot()
	if snapshot.Status != TaskRunning {
		t.Fatalf("resumed status = %s, want %s", snapshot.Status, TaskRunning)
	}
	if snapshot.Error == "" {
		t.Fatal("running resume should expose continued sequential transfer status")
	}
}

func TestTaskWaitIfPausedBlocksUntilResume(t *testing.T) {
	task := NewTask("1", Upload, "a", "b")
	task.mark(TaskRunning, nil)
	task.Pause()
	done := make(chan bool, 1)
	go func() {
		done <- task.WaitIfPaused(task.cancel)
	}()
	select {
	case <-done:
		t.Fatal("WaitIfPaused returned before resume")
	case <-time.After(150 * time.Millisecond):
	}
	task.Resume()
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("WaitIfPaused returned false after resume")
		}
	case <-time.After(time.Second):
		t.Fatal("WaitIfPaused did not return after resume")
	}
}

func TestTempPaths(t *testing.T) {
	if got := TempRemotePath("/var/www/index.html", "abc"); got != "/var/www/.index.html.vssh.tmp.abc" {
		t.Fatalf("remote temp path = %q", got)
	}
	if got := TempLocalPath("/tmp/index.html", "abc"); got != "/tmp/.index.html.vssh.tmp.abc" {
		t.Fatalf("local temp path = %q", got)
	}
}
