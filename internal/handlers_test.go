package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*httptest.Server, *TaskManager) {
	mgr := NewManager(5, 2, []string{".txt"})
	api := &API{Manager: mgr}
	r := chi.NewRouter()
	r.Post("/tasks", api.CreateTask)
	r.Post("/tasks/links", api.AddLink)
	r.Get("/tasks/status/*", api.GetStatus)
	r.Get("/download/*", api.Download)
	ts := httptest.NewServer(r)
	return ts, mgr
}

func TestCreateTask(t *testing.T) {
	ts, mgr := setupTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/tasks", "application/json", nil)
	if err != nil {
		t.Fatalf("create task request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	id := out["task_id"]
	if id == "" {
		t.Fatal("empty task_id")
	}
	if _, err := mgr.Status(id); err != nil {
		t.Fatalf("task not stored: %v", err)
	}
}

func TestAddLinksAndStatus(t *testing.T) {
	// file server returning simple txt
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer fileSrv.Close()

	ts, _ := setupTestServer()
	defer ts.Close()

	// create task
	resp, err := http.Post(ts.URL+"/tasks", "application/json", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id := out["task_id"]

	// add first link
	body, _ := json.Marshal(map[string]string{"task_id": id, "url": fileSrv.URL + "/f1.txt"})
	resp, err = http.Post(ts.URL+"/tasks/links", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("add link1: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// status should be pending
	stResp, err := http.Get(ts.URL + "/tasks/status/" + id)
	if err != nil {
		t.Fatalf("status request: %v", err)
	}
	var status map[string]interface{}
	if err := json.NewDecoder(stResp.Body).Decode(&status); err != nil {
		t.Fatalf("status decode: %v", err)
	}
	if status["status"] != string(StatusPending) {
		t.Fatalf("expected pending, got %v", status["status"])
	}
	stResp.Body.Close()

	// add second link to trigger processing
	body, _ = json.Marshal(map[string]string{"task_id": id, "url": fileSrv.URL + "/f2.txt"})
	resp, err = http.Post(ts.URL+"/tasks/links", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("add link2: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// wait for completion
	for i := 0; i < 40; i++ {
		stResp, err = http.Get(ts.URL + "/tasks/status/" + id)
		if err != nil {
			t.Fatalf("status request: %v", err)
		}
		status = map[string]interface{}{}
		if err := json.NewDecoder(stResp.Body).Decode(&status); err != nil {
			t.Fatalf("decode: %v", err)
		}
		stResp.Body.Close()
		if status["status"] == string(StatusComplete) {
			if status["archive_url"] == nil {
				t.Fatalf("archive_url missing")
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("task did not complete")
}
