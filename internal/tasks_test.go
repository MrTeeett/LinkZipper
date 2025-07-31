package internal

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestCreateUniqueIDs(t *testing.T) {
	mgr := NewManager(1, 1, []string{".txt"})
	const total = 100
	idsCh := make(chan string, total)
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := mgr.Create()
			if err != nil {
				t.Errorf("create task: %v", err)
				return
			}
			idsCh <- id
		}()
	}
	wg.Wait()
	close(idsCh)
	seen := make(map[string]struct{}, total)
	for id := range idsCh {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id: %s", id)
		}
		seen[id] = struct{}{}
	}
}

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

func TestAddURLDuplicate(t *testing.T) {
	mgr := NewManager(1, 3, []string{".txt"})
	id, err := mgr.Create()
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	url := "http://example.com/file.txt"
	if err := mgr.AddURL(id, url); err != nil {
		t.Fatalf("add url: %v", err)
	}
	if err := mgr.AddURL(id, url); err == nil || err.Error() != "this link already exists" {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	task, _ := mgr.Status(id)
	if len(task.Urls) != 1 {
		t.Fatalf("expected 1 url, got %d", len(task.Urls))
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
	if id1 == id2 {
		t.Fatal("duplicate task ids")
	}
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

func TestProcessSkipsInvalidLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok.txt" {
			w.Write([]byte("ok"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	mgr := NewManager(1, 3, []string{".txt"})
	id, _ := mgr.Create()
	okURL := srv.URL + "/ok.txt"
	bad1 := srv.URL + "/bad1.txt"
	bad2 := srv.URL + "/bad2.txt"

	mgr.AddURL(id, okURL)
	mgr.AddURL(id, bad1)
	mgr.AddURL(id, bad2)

	for i := 0; i < 50; i++ {
		task, _ := mgr.Status(id)
		if task.Status == StatusComplete {
			if len(task.Errors) != 2 {
				t.Fatalf("expected 2 errors, got %d", len(task.Errors))
			}
			if _, ok := task.Errors[bad1]; !ok {
				t.Errorf("missing error for bad1")
			}
			if _, ok := task.Errors[bad2]; !ok {
				t.Errorf("missing error for bad2")
			}
			zr, err := zip.OpenReader(task.ZipPath)
			if err != nil {
				t.Fatalf("open zip: %v", err)
			}
			if len(zr.File) != 1 {
				t.Fatalf("expected 1 file in archive, got %d", len(zr.File))
			}
			zr.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("task not completed")
}
