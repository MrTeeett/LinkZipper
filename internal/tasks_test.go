package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAddURLWithQuery(t *testing.T) {
	mgr := NewManager(1, 2, []string{".txt"})
	id, err := mgr.Create()
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := mgr.AddURL(id, "http://example.com/file.txt?token=abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddURLWithQueryInvalidExt(t *testing.T) {
	mgr := NewManager(1, 2, []string{".txt"})
	id, err := mgr.Create()
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := mgr.AddURL(id, "http://example.com/file.exe?download=1"); err == nil {
		t.Fatal("expected error for invalid extension")
	}
}

func TestForceZip(t *testing.T) {
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer fileSrv.Close()

	mgr := NewManager(1, 3, []string{".txt"})
	id, _ := mgr.Create()
	if err := mgr.AddURL(id, fileSrv.URL+"/f1.txt"); err != nil {
		t.Fatalf("add url: %v", err)
	}
	if err := mgr.ForceZip(id); err != nil {
		t.Fatalf("force zip: %v", err)
	}
	for i := 0; i < 50; i++ {
		task, _ := mgr.Status(id)
		if task.Status == StatusComplete {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("task not completed")
}

func TestListAndDelete(t *testing.T) {
	mgr := NewManager(2, 2, []string{".txt"})
	id1, _ := mgr.Create()
	id2, _ := mgr.Create()
	tasks := mgr.List()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if err := mgr.Delete(id1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := mgr.Status(id1); err == nil {
		t.Fatal("expected error for deleted task")
	}
	tasks = mgr.List()
	if len(tasks) != 1 || tasks[0].ID != id2 {
		t.Fatal("unexpected tasks after delete")
	}
}
