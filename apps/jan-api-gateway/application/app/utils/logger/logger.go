// logger/logger.go
package logger

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	Logger *logrus.Logger
	once   sync.Once
)

// GetLogger returns the singleton logger instance
func GetLogger() *logrus.Logger {
	once.Do(func() {
		Logger = logrus.New()
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
		Logger.SetOutput(os.Stdout)
		Logger.SetLevel(logrus.InfoLevel)
		Logger.SetOutput(os.Stdout)
	})
	return Logger
}
