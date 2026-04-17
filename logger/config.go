package logger

import (
	"io"
	"os"
)

type Option func(*Logger)

func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

func WithLevelString(s string) Option {
	return func(l *Logger) {
		l.level = ParseLevel(s)
	}
}

func WithModule(module string) Option {
	return func(l *Logger) {
		l.module = module
	}
}

func WithWriters(writers ...io.Writer) Option {
	return func(l *Logger) {
		l.writers = writers
	}
}

func WithColorize(colorize bool) Option {
	return func(l *Logger) {
		l.colorize = colorize
	}
}

func WithFields(fields ...any) Option {
	return func(l *Logger) {
		l.fields = fields
	}
}

type Config struct {
	Level       string `json:"level"`
	Module      string `json:"module"`
	Colorize    bool   `json:"colorize"`
	FileOutput  bool   `json:"file_output"`
	FilePath    string `json:"file_path"`
	MaxSize     int64  `json:"max_size"`
	MaxBackups  int    `json:"max_backups"`
	MaxAge      int    `json:"max_age"`
	RotateByDate bool  `json:"rotate_by_date"`
}

func DefaultConfig() Config {
	return Config{
		Level:       "INFO",
		Module:      "",
		Colorize:    true,
		FileOutput:  false,
		FilePath:    "./logs/app.log",
		MaxSize:     100 * 1024 * 1024,
		MaxBackups:  5,
		MaxAge:      30,
		RotateByDate: false,
	}
}

func NewFromConfig(cfg Config) *Logger {
	opts := []Option{
		WithLevelString(cfg.Level),
		WithModule(cfg.Module),
		WithColorize(cfg.Colorize),
	}

	if cfg.FileOutput {
		fw := NewRotatingFileWriter(cfg.FilePath, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge, cfg.RotateByDate)
		opts = append(opts, WithWriters(os.Stdout, fw))
	}

	return New(opts...)
}
