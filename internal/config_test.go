package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgContent := []byte("server:\n  port: 9090\nlimits:\n  maxTasks: 5\n  maxFilesPerTask: 3\n  allowedExtensions:\n    - \".txt\"\nlogging:\n  level: debug\n  file: app.log\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), cfgContent, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	t.Cleanup(viper.Reset)
	cfg := Load()

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Limits.MaxTasks != 5 || cfg.Limits.MaxFilesPerTask != 3 {
		t.Fatalf("unexpected limits: %+v", cfg.Limits)
	}
	if cfg.Logging.Level != "debug" || cfg.Logging.File != "app.log" {
		t.Fatalf("unexpected logging config: %+v", cfg.Logging)
	}
	if len(cfg.Limits.AllowedExts) != 1 || cfg.Limits.AllowedExts[0] != ".txt" {
		t.Fatalf("unexpected allowed extensions: %+v", cfg.Limits.AllowedExts)
	}
}