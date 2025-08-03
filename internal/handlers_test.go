package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*httptest.Server, *TaskManager) {
	return setupTestServerLimits(5, 2)
}

func setupTestServerLimits(maxTasks, maxFiles int) (*httptest.Server, *TaskManager) {
	mgr := NewManager(maxTasks, maxFiles, []string{".txt"})
	api := &API{Manager: mgr}
	r := chi.NewRouter()
	r.Post("/tasks", api.CreateTask)
	r.Get("/tasks/list", api.ListTasks)
	r.Post("/tasks/links", api.AddLink)
	r.Post("/tasks/zip", api.ForceZip)
	r.Delete("/tasks/delete/*", api.DeleteTask)
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

func TestCreateTaskUniqueIDs(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	const total = 100
	idsCh := make(chan string, total)
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(ts.URL+"/tasks", "application/json", nil)
			if err != nil {
				t.Errorf("create task: %v", err)
				return
			}
			var out map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Errorf("decode: %v", err)
				resp.Body.Close()
				return
			}
			resp.Body.Close()
			idsCh <- out["task_id"]
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

func TestAddLinksAndStatus(t *testing.T) {
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer fileSrv.Close()

	ts, _ := setupTestServer()
	defer ts.Close()

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

	body, _ := json.Marshal(map[string]string{"task_id": id, "url": fileSrv.URL + "/f1.txt"})
	resp, err = http.Post(ts.URL+"/tasks/links", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("add link1: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

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

	body, _ = json.Marshal(map[string]string{"task_id": id, "url": fileSrv.URL + "/f2.txt"})
	resp, err = http.Post(ts.URL+"/tasks/links", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("add link2: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

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

func TestCreateTaskServerBusy(t *testing.T) {
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("ok"))
	}))
	defer fileSrv.Close()

	ts, _ := setupTestServerLimits(1, 1)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/tasks", "application/json", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resp.Body.Close()
	id := out["task_id"]

	body, _ := json.Marshal(map[string]string{"task_id": id, "url": fileSrv.URL + "/f.txt"})
	resp, err = http.Post(ts.URL+"/tasks/links", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("add link: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Post(ts.URL+"/tasks", "application/json", nil)
	if err != nil {
		t.Fatalf("create second task: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
}

func TestListDeleteAndForceZip(t *testing.T) {
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer fileSrv.Close()

	ts, _ := setupTestServer()
	defer ts.Close()

	resp, _ := http.Post(ts.URL+"/tasks", "application/json", nil)
	var out1 map[string]string
	json.NewDecoder(resp.Body).Decode(&out1)
	resp.Body.Close()
	resp, _ = http.Post(ts.URL+"/tasks", "application/json", nil)
	var out2 map[string]string
	json.NewDecoder(resp.Body).Decode(&out2)
	resp.Body.Close()
	if out1["task_id"] == out2["task_id"] {
		t.Fatal("duplicate task ids")
	}
	listResp, err := http.Get(ts.URL + "/tasks/list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	listResp.Body.Close()
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}

	body, _ := json.Marshal(map[string]string{"task_id": out1["task_id"], "url": fileSrv.URL + "/f.txt"})
	http.Post(ts.URL+"/tasks/links", "application/json", bytes.NewReader(body))

	body, _ = json.Marshal(map[string]string{"task_id": out1["task_id"]})
	resp, err = http.Post(ts.URL+"/tasks/zip", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("force zip request failed")
	}
	resp.Body.Close()

	for i := 0; i < 40; i++ {
		st, _ := http.Get(ts.URL + "/tasks/status/" + out1["task_id"])
		var status map[string]interface{}
		json.NewDecoder(st.Body).Decode(&status)
		st.Body.Close()
		if status["status"] == string(StatusComplete) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/tasks/delete/"+out2["task_id"], nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("delete request failed")
	}
	resp.Body.Close()

	listResp, _ = http.Get(ts.URL + "/tasks/list")
	list = []map[string]interface{}{}
	json.NewDecoder(listResp.Body).Decode(&list)
	listResp.Body.Close()
	if len(list) != 1 {
		t.Fatalf("expected 1 task after delete, got %d", len(list))
	}
}
