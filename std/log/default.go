package log

import "os"

var defaultLogger *Logger = NewText(os.Stderr)

// Default returns the default logger.
func Default() *Logger {
	return defaultLogger
}

// Trace level message.
func Trace(msg string, v ...any) {
	defaultLogger.log(nil, msg, LevelTrace, v...)
}

// Debug level message.
func Debug(msg string, v ...any) {
	defaultLogger.log(nil, msg, LevelDebug, v...)
}

// Info level message.
func Info(msg string, v ...any) {
	defaultLogger.Info(nil, msg, v...)
}

// Warn level message.
func Warn(msg string, v ...any) {
	defaultLogger.Warn(nil, msg, v...)
}

// Error level message.
func Error(msg string, v ...any) {
	defaultLogger.Error(nil, msg, v...)
}

// Fatal level message, followed by an exit.
func Fatal(msg string, v ...any) {
	defaultLogger.Fatal(nil, msg, v...)
}

// Tracef level formatted message.
func Tracef(msg string, v ...any) {
	defaultLogger.Trace(nil, msg, v...)
}

// Debugf level formatted message.
func Debugf(msg string, v ...any) {
	defaultLogger.Debug(nil, msg, v...)
}

// Infof level formatted message.
func Infof(msg string, v ...any) {
	defaultLogger.Info(nil, msg, v...)
}

// Warnf level formatted message.
func Warnf(msg string, v ...any) {
	defaultLogger.Warn(nil, msg, v...)
}

// Errorf level formatted message.
func Errorf(msg string, v ...any) {
	defaultLogger.Error(nil, msg, v...)
}

// Fatalf level formatted message, followed by an exit.
func Fatalf(msg string, v ...any) {
	defaultLogger.Fatal(nil, msg, v...)
}

// HasTrace returns if trace level is enabled.
func HasTrace() bool {
	return defaultLogger.level <= LevelTrace
}
