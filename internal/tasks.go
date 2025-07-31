package internal

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type TaskStatus string

const (
	StatusPending  TaskStatus = "pending"
	StatusProcessing TaskStatus = "processing"
	StatusComplete TaskStatus = "complete"
)

type Task struct {
	ID          string
	Urls        []string
	Errors      map[string]string
	ZipPath     string
	Status      TaskStatus
	createdAt   time.Time
}

var idCounter uint64

// TaskManager хранит задачи в памяти
type TaskManager struct {
	mu        sync.Mutex
	tasks     map[string]*Task
	completed map[string]*Task
	inProcess int
	maxTasks  int
	maxFiles  int
	exts      map[string]struct{}
}

func NewManager(maxTasks, maxFiles int, allowedExts []string) *TaskManager {
	exts := make(map[string]struct{}, len(allowedExts))
	for _, e := range allowedExts {
		exts[e] = struct{}{}
	}
	return &TaskManager{
		tasks:     make(map[string]*Task),
		completed: make(map[string]*Task),
		maxTasks:  maxTasks,
		maxFiles:  maxFiles,
		exts:      exts,
	}
}

func (m *TaskManager) Create() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.inProcess >= m.maxTasks {
		err := errors.New("server busy: max tasks reached")
		Logger.WithError(err).Error("create task failed")
		return "", err
	}

	id := fmt.Sprintf("task-%d", atomic.AddUint64(&idCounter, 1))
	m.tasks[id] = &Task{ID: id, Urls: []string{}, Errors: make(map[string]string), Status: StatusPending, createdAt: time.Now()}
	Logger.WithField("task_id", id).Info("task created")
	return id, nil
}

func (m *TaskManager) AddURL(id, url string) error {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		if _, done := m.completed[id]; done {
			err := errors.New("task already completed")
			Logger.WithError(err).WithField("task_id", id).Error("add url failed")
			return err
		}
		err := errors.New("task not found")
		Logger.WithError(err).WithField("task_id", id).Error("add url failed")
		return err
	}
	if task.Status != StatusPending {
		m.mu.Unlock()
		err := errors.New("task already processing")
		Logger.WithError(err).WithField("task_id", id).Error("add url failed")
		return err
	}
	if len(task.Urls) >= m.maxFiles {
		m.mu.Unlock()
		err := errors.New("max files per task reached")
		Logger.WithError(err).WithField("task_id", id).Error("add url failed")
		return err
	}
	parsed, err := neturl.Parse(url)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		m.mu.Unlock()
		err := errors.New("invalid URL")
		Logger.WithError(err).WithField("task_id", id).Error("add url failed")
		return err
	}
	ext := filepath.Ext(parsed.Path)
	if _, allowed := m.exts[ext]; !allowed {
		m.mu.Unlock()
		err := fmt.Errorf("extension %s not allowed", ext)
		Logger.WithError(err).WithField("task_id", id).Error("add url failed")
		return err
	}
	task.Urls = append(task.Urls, url)
	shouldZip := len(task.Urls) == m.maxFiles
	if shouldZip {
		if m.inProcess >= m.maxTasks {
			m.mu.Unlock()
			err := errors.New("server busy: max tasks reached")
			Logger.WithError(err).WithField("task_id", id).Error("add url failed")
			return err
		}
		m.inProcess++
		task.Status = StatusProcessing
	}
	m.mu.Unlock()

	if shouldZip {
		Logger.WithField("task_id", id).Info("processing started")
		go m.process(task)
	} else {
		Logger.WithFields(logrus.Fields{"task_id": id, "url": url}).Info("url added")
	}
	return nil
}

func (m *TaskManager) ForceZip(id string) error {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		if _, done := m.completed[id]; done {
			err := errors.New("task already completed")
			Logger.WithError(err).WithField("task_id", id).Error("force zip failed")
			return err
		}
		err := errors.New("task not found")
		Logger.WithError(err).WithField("task_id", id).Error("force zip failed")
		return err
	}
	if task.Status != StatusPending {
		m.mu.Unlock()
		err := errors.New("task already processing")
		Logger.WithError(err).WithField("task_id", id).Error("force zip failed")
		return err
	}
	if len(task.Urls) == 0 {
		m.mu.Unlock()
		err := errors.New("no files to archive")
		Logger.WithError(err).WithField("task_id", id).Error("force zip failed")
		return err
	}
	if m.inProcess >= m.maxTasks {
		m.mu.Unlock()
		err := errors.New("server busy: max tasks reached")
		Logger.WithError(err).WithField("task_id", id).Error("force zip failed")
		return err
	}
	m.inProcess++
	task.Status = StatusProcessing
	m.mu.Unlock()

	Logger.WithField("task_id", id).Info("manual processing started")
	go m.process(task)
	return nil
}

func (m *TaskManager) process(task *Task) {
	tmpDir := os.TempDir()
	zipName := fmt.Sprintf("%s.zip", task.ID)
	zipPath := filepath.Join(tmpDir, zipName)
	f, _ := os.Create(zipPath)
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, url := range task.Urls {
		resp, err := http.Get(url)
		if err != nil {
			task.Errors[url] = err.Error()
			Logger.WithError(err).WithField("url", url).Error("download failed")
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			task.Errors[url] = fmt.Sprintf("status %d", resp.StatusCode)
			Logger.WithField("url", url).Errorf("status %d", resp.StatusCode)
			return
		}
		fname := filepath.Base(url)
		w, _ := zw.Create(fname)
		if _, err := io.Copy(w, resp.Body); err != nil {
			task.Errors[url] = err.Error()
			Logger.WithError(err).WithField("url", url).Error("write failed")
		} else {
			Logger.WithFields(logrus.Fields{"task_id": task.ID, "file": fname}).Info("file added")
		}
	}
	task.ZipPath = zipPath
	task.Status = StatusComplete
	m.mu.Lock()
	m.inProcess--
	delete(m.tasks, task.ID)
	m.completed[task.ID] = task
	Logger.WithField("task_id", task.ID).Info("task completed")
	m.mu.Unlock()
}

func (m *TaskManager) Status(id string) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	if task, ok := m.completed[id]; ok {
		return task, nil
	}
	return nil, errors.New("task not found")
}

func (m *TaskManager) List() []*Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Task, 0, len(m.tasks)+len(m.completed))
	for _, t := range m.tasks {
		out = append(out, t)
	}
	for _, t := range m.completed {
		out = append(out, t)
	}
	return out
}

func (m *TaskManager) Delete(id string) error {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if ok {
		if task.Status == StatusProcessing {
			m.mu.Unlock()
			err := errors.New("task in progress")
			Logger.WithError(err).WithField("task_id", id).Error("delete task failed")
			return err
		}
		delete(m.tasks, id)
		Logger.WithField("task_id", id).Info("task deleted")
		m.mu.Unlock()
		return nil
	}
	task, ok = m.completed[id]
	if ok {
		delete(m.completed, id)
		if task.ZipPath != "" {
			os.Remove(task.ZipPath)
		}
		Logger.WithField("task_id", id).Info("task deleted")
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()
	err := errors.New("task not found")
	Logger.WithError(err).WithField("task_id", id).Error("delete task failed")
	return err
}