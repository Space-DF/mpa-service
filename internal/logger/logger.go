package logger

import (
	"log"
	"os"
	"strings"
)

type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	logLevel    string
}

func New(logLevel string) *Logger {
	flags := log.Ldate | log.Ltime | log.Lshortfile

	return &Logger{
		debugLogger: log.New(os.Stdout, "DEBUG: ", flags),
		infoLogger:  log.New(os.Stdout, "INFO: ", flags),
		warnLogger:  log.New(os.Stdout, "WARN: ", flags),
		errorLogger: log.New(os.Stderr, "ERROR: ", flags),
		logLevel:    strings.ToLower(logLevel),
	}
}

func (l *Logger) shouldLog(level string) bool {
	switch l.logLevel {
	case "debug":
		return true
	case "info":
		return level != "debug"
	case "warn":
		return level != "debug" && level != "info"
	case "error":
		return level == "error"
	default:
		return level != "debug" // Default to info level
	}
}

func (l *Logger) Debug(v ...interface{}) {
	if l.shouldLog("debug") {
		l.debugLogger.Println(v...)
	}
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.shouldLog("debug") {
		l.debugLogger.Printf(format, v...)
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.shouldLog("info") {
		l.infoLogger.Println(v...)
	}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.shouldLog("info") {
		l.infoLogger.Printf(format, v...)
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.shouldLog("warn") {
		l.warnLogger.Println(v...)
	}
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.shouldLog("warn") {
		l.warnLogger.Printf(format, v...)
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.shouldLog("error") {
		l.errorLogger.Println(v...)
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.shouldLog("error") {
		l.errorLogger.Printf(format, v...)
	}
}
