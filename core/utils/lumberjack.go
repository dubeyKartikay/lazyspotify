package utils

import (
	"path/filepath"
	"github.com/natefinch/lumberjack"
)
func NewLumberjackLogger(logFileName string) *lumberjack.Logger {
	configDir := SafeGetConfigDir()
	logFile := filepath.Join(configDir, logFileName)
	lumberjackLogger := &lumberjack.Logger{
        Filename:   logFile,
        MaxSize:    10, // Max size in megabytes before log is rotated
        MaxBackups: 3,  // Max number of old log files to retain
        MaxAge:     28, // Max number of days to retain old log files
        Compress:   true, // Compress old log files
	}
  return lumberjackLogger
}
