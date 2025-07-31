package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInitLoggerSetsLevel(t *testing.T) {
	prevLevel := Logger.GetLevel()
	prevOut := Logger.Out
	defer func() {
		Logger.SetLevel(prevLevel)
		Logger.SetOutput(prevOut)
	}()
	defer CloseLogger()
	InitLogger("debug", "")
	if Logger.GetLevel() != logrus.DebugLevel {
		t.Fatalf("expected debug level, got %v", Logger.GetLevel())
	}
}

func TestInitLoggerInvalidLevelDefaultsToInfo(t *testing.T) {
	prevLevel := Logger.GetLevel()
	prevOut := Logger.Out
	defer func() {
		Logger.SetLevel(prevLevel)
		Logger.SetOutput(prevOut)
	}()
	defer CloseLogger()
	InitLogger("invalid", "")
	if Logger.GetLevel() != logrus.InfoLevel {
		t.Fatalf("expected info level, got %v", Logger.GetLevel())
	}
}

func TestInitLoggerWithFile(t *testing.T) {
	prevLevel := Logger.GetLevel()
	prevOut := Logger.Out
	defer func() {
		Logger.SetLevel(prevLevel)
		Logger.SetOutput(prevOut)
	}()
	defer CloseLogger()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.log")

	InitLogger("info", filePath)
	Logger.Info("hello world")

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello world") {
		t.Fatalf("log file does not contain expected message")
	}
}