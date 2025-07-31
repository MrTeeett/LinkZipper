package internal

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	Logger = logrus.New()
	logFile *os.File
)

func InitLogger(level, file string) {
	if lvl, err := logrus.ParseLevel(level); err == nil {
		Logger.SetLevel(lvl)
	} else {
		Logger.SetLevel(logrus.InfoLevel)
	}
	CloseLogger()
	Logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			Logger.SetOutput(io.MultiWriter(os.Stdout, f))
		} else {
			Logger.Warnf("Failed to open log file %s: %v", file, err)
		}
	}
}

func CloseLogger() {
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
	Logger.SetOutput(os.Stdout)
}
