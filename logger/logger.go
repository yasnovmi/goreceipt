package logger

import (
	"os"
	"path"

	. "github.com/yasnov/goreceipt/config"

	"github.com/sirupsen/logrus"
)

type ProvidersLogger struct {
	*logrus.Logger
}

func CreateNewProvidersLogger() *ProvidersLogger {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	dirPath := path.Join(Config.LogPath, "loader/")
	f, err := NewLogFile(path.Join(dirPath, "providers") + ".log")
	if err == nil {
		log.SetOutput(f)
	}
	return &ProvidersLogger{log}
}

func CreateDBLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	dirPath := path.Join(Config.LogPath, "db/")
	f, err := NewLogFile(path.Join(dirPath, "database") + ".log")
	if err == nil {
		logger.SetOutput(f)
	}
	return logger
}

func NewLogFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return f, nil
	}
	return f, nil
}
