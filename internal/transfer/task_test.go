package transfer

import (
	"context"
	"os"
	"path/filepath"
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

func TestManagerCancelAllAndActiveCount(t *testing.T) {
	manager := NewManager()
	first := NewTask("1", Upload, "a", "b")
	second := NewTask("2", Download, "c", "d")
	manager.Add(first)
	manager.Add(second)
	if got := manager.ActiveCount(); got != 2 {
		t.Fatalf("active count = %d, want 2", got)
	}
	if canceled := manager.CancelAll(); canceled != 2 {
		t.Fatalf("canceled = %d, want 2", canceled)
	}
	if got := manager.ActiveCount(); got != 0 {
		t.Fatalf("active count after cancel = %d, want 0", got)
	}
	if first.Snapshot().Status != TaskCanceled || second.Snapshot().Status != TaskCanceled {
		t.Fatalf("tasks were not canceled: %s %s", first.Snapshot().Status, second.Snapshot().Status)
	}
}

func TestManagerWaitReturnsWithoutRunningTasks(t *testing.T) {
	manager := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Wait(ctx); err != nil {
		t.Fatalf("wait returned error: %v", err)
	}
}

func TestManagerCancelAndRemoveQueuedTask(t *testing.T) {
	manager := NewManager()
	task := NewTask("1", Upload, "a", "b")
	manager.Add(task)
	if err := manager.CancelAndRemove(task.ID); err != nil {
		t.Fatal(err)
	}
	if got := manager.Tasks(); len(got) != 0 {
		t.Fatalf("tasks = %d, want 0", len(got))
	}
}

func TestCleanupTaskTempRemovesLocalDownloadTemp(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	task := NewTask("abc", Download, "/remote/target.txt", target)
	tempPath := TempLocalPath(target, task.ID)
	if err := os.WriteFile(tempPath, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cleanupTaskTemp(nil, task); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temp path still exists or unexpected stat error: %v", err)
	}
}

func TestTaskTempPathSnapshot(t *testing.T) {
	task := NewTask("1", Upload, "a", "b")
	task.SetTempPath("/tmp/.b.vssh.tmp.1")
	if got := task.Snapshot().TempPath; got != "/tmp/.b.vssh.tmp.1" {
		t.Fatalf("temp path = %q", got)
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
