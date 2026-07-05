package transfer

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/pkg/sftp"
)

type Manager struct {
	mu          sync.Mutex
	tasks       map[string]*Task
	jobs        map[string]*job
	concurrency int
	running     int
	wg          sync.WaitGroup
}

type job struct {
	ctx    context.Context
	client *sftp.Client
	task   *Task
}

func NewManager() *Manager {
	return &Manager{tasks: map[string]*Task{}, jobs: map[string]*job{}, concurrency: 1}
}

func (m *Manager) SetConcurrency(n int) {
	if n <= 0 {
		n = 1
	}
	m.mu.Lock()
	m.concurrency = n
	m.mu.Unlock()
	m.schedule()
}

func (m *Manager) Add(task *Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID] = task
}

func (m *Manager) Tasks() []*Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		cp := t.Snapshot()
		out = append(out, &cp)
	}
	return out
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	task := m.tasks[id]
	m.mu.Unlock()
	if task == nil {
		return fmt.Errorf("task %q not found", id)
	}
	task.Cancel()
	m.schedule()
	return nil
}

func (m *Manager) CancelAndRemove(id string) error {
	m.mu.Lock()
	task := m.tasks[id]
	job := m.jobs[id]
	m.mu.Unlock()
	if task == nil {
		return fmt.Errorf("task %q not found", id)
	}
	task.Cancel()
	if job != nil {
		_ = cleanupTaskTemp(job.client, task)
	}
	m.mu.Lock()
	delete(m.jobs, id)
	delete(m.tasks, id)
	m.mu.Unlock()
	m.schedule()
	return nil
}

func (m *Manager) CancelAll() int {
	m.mu.Lock()
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		snapshot := task.Snapshot()
		if snapshot.Status == TaskQueued || snapshot.Status == TaskRunning || snapshot.Status == TaskPaused {
			tasks = append(tasks, task)
		}
	}
	m.mu.Unlock()
	for _, task := range tasks {
		task.Cancel()
	}
	m.schedule()
	return len(tasks)
}

func (m *Manager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	active := 0
	for _, task := range m.tasks {
		snapshot := task.Snapshot()
		if snapshot.Status == TaskQueued || snapshot.Status == TaskRunning || snapshot.Status == TaskPaused {
			active++
		}
	}
	return active
}

func (m *Manager) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (m *Manager) Pause(id string) error {
	m.mu.Lock()
	task := m.tasks[id]
	m.mu.Unlock()
	if task == nil {
		return fmt.Errorf("task %q not found", id)
	}
	task.Pause()
	return nil
}

func (m *Manager) Resume(id string) error {
	m.mu.Lock()
	task := m.tasks[id]
	m.mu.Unlock()
	if task == nil {
		return fmt.Errorf("task %q not found", id)
	}
	task.Resume()
	m.schedule()
	return nil
}

func (m *Manager) Start(ctx context.Context, client *sftp.Client, task *Task) {
	m.Add(task)
	m.mu.Lock()
	m.jobs[task.ID] = &job{ctx: ctx, client: client, task: task}
	m.mu.Unlock()
	m.schedule()
}

func (m *Manager) schedule() {
	m.mu.Lock()
	if m.concurrency <= 0 {
		m.concurrency = 1
	}
	var ready []*job
	for m.running < m.concurrency {
		var next *job
		for _, candidate := range m.jobs {
			snapshot := candidate.task.Snapshot()
			if snapshot.Status == TaskQueued {
				next = candidate
				break
			}
		}
		if next == nil {
			break
		}
		next.task.mark(TaskRunning, nil)
		m.running++
		m.wg.Add(1)
		ready = append(ready, next)
	}
	m.mu.Unlock()
	for _, job := range ready {
		go m.run(job)
	}
}

func (m *Manager) run(job *job) {
	defer m.wg.Done()
	progress := func(done, total int64) {
		job.task.SetProgress(done, total)
	}
	var err error
	switch job.task.Direction {
	case Upload:
		err = AtomicUpload(job.client, job.task.SourcePath, job.task.TargetPath, job.task.ID, progress, job.task.SetTempPath, job.task.cancel, job.task.WaitIfPaused)
	case Download:
		err = AtomicDownload(job.client, job.task.SourcePath, job.task.TargetPath, job.task.ID, progress, job.task.SetTempPath, job.task.cancel, job.task.WaitIfPaused)
	default:
		err = fmt.Errorf("unsupported transfer direction %q", job.task.Direction)
	}
	select {
	case <-job.ctx.Done():
		job.task.mark(TaskCanceled, job.ctx.Err())
	default:
		if err != nil {
			if job.task.IsCanceled() {
				job.task.mark(TaskCanceled, err)
			} else {
				job.task.mark(TaskFailed, err)
			}
		} else {
			job.task.mark(TaskSucceeded, nil)
		}
	}
	m.mu.Lock()
	m.running--
	delete(m.jobs, job.task.ID)
	if job.task.Snapshot().Status == TaskCanceled {
		delete(m.tasks, job.task.ID)
	}
	m.mu.Unlock()
	m.schedule()
}

func cleanupTaskTemp(client *sftp.Client, task *Task) error {
	if task == nil {
		return nil
	}
	snapshot := task.Snapshot()
	tempPath := snapshot.TempPath
	if tempPath == "" {
		switch snapshot.Direction {
		case Upload:
			tempPath = TempRemotePath(snapshot.TargetPath, snapshot.ID)
		case Download:
			tempPath = TempLocalPath(snapshot.TargetPath, snapshot.ID)
		}
	}
	if tempPath == "" {
		return nil
	}
	if snapshot.Direction == Upload {
		if client == nil {
			return nil
		}
		return client.Remove(tempPath)
	}
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
