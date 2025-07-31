package internal

import "testing"

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