package logger

import (
	"log"
	"os"
)

type Logger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
}

func New(logLevel string) *Logger {
	flags := log.Ldate | log.Ltime | log.Lshortfile
	
	return &Logger{
		infoLogger:  log.New(os.Stdout, "INFO: ", flags),
		warnLogger:  log.New(os.Stdout, "WARN: ", flags),
		errorLogger: log.New(os.Stderr, "ERROR: ", flags),
	}
}

func (l *Logger) Info(v ...interface{}) {
	l.infoLogger.Println(v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

func (l *Logger) Warn(v ...interface{}) {
	l.warnLogger.Println(v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.warnLogger.Printf(format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.errorLogger.Println(v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}