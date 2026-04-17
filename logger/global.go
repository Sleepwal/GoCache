package logger

import (
	"io"
	"sync"
)

var (
	global     *Logger
	globalOnce sync.Once
)

func init() {
	globalOnce.Do(func() {
		global = New()
	})
}

func Init(opts ...Option) {
	globalOnce.Do(func() {})
	global.Close()
	global = New(opts...)
}

func InitFromConfig(cfg Config) {
	globalOnce.Do(func() {})
	global.Close()
	global = NewFromConfig(cfg)
}

func SetLevel(level Level) {
	global.SetLevel(level)
}

func GetLevel() Level {
	return global.GetLevel()
}

func AddWriter(w io.Writer) {
	global.AddWriter(w)
}

func With(fields ...any) *Logger {
	return global.With(fields...)
}

func Debug(msg string, fields ...any) {
	global.Debug(msg, fields...)
}

func Info(msg string, fields ...any) {
	global.Info(msg, fields...)
}

func Warn(msg string, fields ...any) {
	global.Warn(msg, fields...)
}

func Error(msg string, fields ...any) {
	global.Error(msg, fields...)
}

func ErrorErr(msg string, err error, fields ...any) {
	global.ErrorErr(msg, err, fields...)
}

func Fatal(msg string, fields ...any) {
	global.Fatal(msg, fields...)
}

func FatalErr(msg string, err error, fields ...any) {
	global.FatalErr(msg, err, fields...)
}

func Debugf(format string, args ...any) {
	global.Debugf(format, args...)
}

func Infof(format string, args ...any) {
	global.Infof(format, args...)
}

func Warnf(format string, args ...any) {
	global.Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	global.Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	global.Fatalf(format, args...)
}

func Close() error {
	return global.Close()
}

func Sync() {
	global.Sync()
}
