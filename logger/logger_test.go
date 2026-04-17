package logger

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func newTestLogger(buf *bytes.Buffer) *Logger {
	return New(
		WithLevel(DEBUG),
		WithWriters(buf),
		WithColorize(false),
	)
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{Level(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"FATAL", FATAL},
		{"fatal", FATAL},
		{"unknown", INFO},
	}
	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.expected {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestLogOutputFormat(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	l.Info("test message")
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("log output should contain level INFO, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("log output should contain message, got: %s", output)
	}

	tsPattern := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}`)
	if !tsPattern.MatchString(output) {
		t.Errorf("log output should contain timestamp in format YYYY-MM-DD HH:MM:SS.mmm, got: %s", output)
	}

	callerPattern := regexp.MustCompile(`\w+\.go:\d+ [\w.]+`)
	if !callerPattern.MatchString(output) {
		t.Errorf("log output should contain file:line function, got: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	l.Debug("debug msg")
	l.Info("info msg")
	l.Warn("warn msg")
	l.Error("error msg")
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Error("should contain DEBUG level")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("should contain INFO level")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("should contain WARN level")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("should contain ERROR level")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := New(
		WithLevel(WARN),
		WithWriters(&buf),
		WithColorize(false),
	)
	defer l.Close()

	l.Debug("should not appear")
	l.Info("should not appear")
	l.Warn("should appear")
	l.Error("should appear")
	l.Sync()

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("DEBUG and INFO logs should be filtered out when level is WARN")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("WARN and ERROR logs should pass through")
	}
}

func TestLogWithFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	l.Info("user logged in", "user_id", 123, "ip", "192.168.1.1")
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "user_id=123") {
		t.Errorf("log should contain structured field user_id=123, got: %s", output)
	}
	if !strings.Contains(output, "ip=192.168.1.1") {
		t.Errorf("log should contain structured field ip=192.168.1.1, got: %s", output)
	}
}

func TestLogWithOddFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	l.Info("test", "key_only")
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "key_only=<missing>") {
		t.Errorf("odd field should show <missing>, got: %s", output)
	}
}

func TestLogErrorErr(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	testErr := errors.New("database connection failed")
	l.ErrorErr("operation failed", testErr, "retry", 3)
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "error=database connection failed") {
		t.Errorf("log should contain error message, got: %s", output)
	}
	if !strings.Contains(output, "retry=3") {
		t.Errorf("log should contain retry field, got: %s", output)
	}
}

func TestLogFormatMethods(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	l.Debugf("debug %d", 1)
	l.Infof("info %s", "hello")
	l.Warnf("warn %v", true)
	l.Errorf("error %f", 3.14)
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "debug 1") {
		t.Error("Debugf should format message")
	}
	if !strings.Contains(output, "info hello") {
		t.Error("Infof should format message")
	}
	if !strings.Contains(output, "warn true") {
		t.Error("Warnf should format message")
	}
	if !strings.Contains(output, "error 3.14") {
		t.Error("Errorf should format message")
	}
}

func TestWithCreatesChild(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	child := l.With("request_id", "abc-123")
	child.Info("processing request")
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "request_id=abc-123") {
		t.Errorf("child logger should include parent fields, got: %s", output)
	}
	if !strings.Contains(output, "processing request") {
		t.Errorf("child logger should log message, got: %s", output)
	}
}

func TestWithModule(t *testing.T) {
	var buf bytes.Buffer
	l := New(
		WithLevel(DEBUG),
		WithWriters(&buf),
		WithColorize(false),
		WithModule("cache"),
	)
	defer l.Close()

	l.Info("cache hit")
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "module=cache") {
		t.Errorf("log should contain module field, got: %s", output)
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	l.SetLevel(ERROR)
	if l.GetLevel() != ERROR {
		t.Error("GetLevel should return ERROR after SetLevel(ERROR)")
	}

	l.Info("should not appear")
	l.Error("should appear")
	l.Sync()

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("INFO should be filtered after SetLevel(ERROR)")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("ERROR should pass through")
	}
}

func TestAddWriter(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	l := New(
		WithLevel(DEBUG),
		WithWriters(&buf1),
		WithColorize(false),
	)
	defer l.Close()

	l.AddWriter(&buf2)
	l.Info("multi writer test")
	l.Sync()

	if buf1.String() == "" {
		t.Error("first writer should receive output")
	}
	if buf2.String() == "" {
		t.Error("second writer should receive output")
	}
}

func TestCloseAndSync(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)

	l.Info("before close")
	l.Sync()

	output := buf.String()
	if !strings.Contains(output, "before close") {
		t.Error("should log before close")
	}

	err := l.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}

func TestShortFile(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/project/logger/logger.go", "logger/logger.go"},
		{"C:\\Users\\dev\\project\\logger\\logger.go", "logger\\logger.go"},
		{"logger.go", "logger.go"},
	}
	for _, tt := range tests {
		if got := shortFile(tt.input); got != tt.expected {
			t.Errorf("shortFile(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestShortFuncName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GoCache/logger.TestLogOutputFormat", "logger.TestLogOutputFormat"},
		{"main.main", "main.main"},
		{"runtime.main", "runtime.main"},
	}
	for _, tt := range tests {
		if got := shortFuncName(tt.input); got != tt.expected {
			t.Errorf("shortFuncName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatFields(t *testing.T) {
	tests := []struct {
		fields   []any
		expected string
	}{
		{[]any{}, ""},
		{[]any{"key", "value"}, "key=value"},
		{[]any{"a", 1, "b", 2}, "a=1 b=2"},
		{[]any{"odd"}, "odd=<missing>"},
	}
	for _, tt := range tests {
		if got := formatFields(tt.fields); got != tt.expected {
			t.Errorf("formatFields(%v) = %q, want %q", tt.fields, got, tt.expected)
		}
	}
}

func TestRotatingFileWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	w := NewRotatingFileWriter(logPath, 100, 3, 0, false)

	msg := "hello logger\n"
	n, err := w.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(msg) {
		t.Errorf("Write returned %d, want %d", n, len(msg))
	}

	w.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "hello logger") {
		t.Errorf("file should contain written content, got: %s", string(content))
	}
}

func TestRotatingFileWriterSizeRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "rotate.log")

	maxSize := int64(50)
	w := NewRotatingFileWriter(logPath, maxSize, 3, 0, false)

	data := strings.Repeat("a", 30)
	for i := 0; i < 5; i++ {
		w.Write([]byte(data + "\n"))
	}
	w.Close()

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	backupCount := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "rotate-") {
			backupCount++
		}
	}

	if backupCount == 0 {
		t.Error("expected at least one rotated backup file")
	}
}

func TestRotatingFileWriterMaxBackups(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "maxback.log")

	maxSize := int64(20)
	maxBackups := 2
	w := NewRotatingFileWriter(logPath, maxSize, maxBackups, 0, false)

	data := strings.Repeat("x", 15)
	for i := 0; i < 10; i++ {
		w.Write([]byte(data + "\n"))
		time.Sleep(1 * time.Millisecond)
	}
	w.Close()

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	backupCount := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "maxback-") {
			backupCount++
		}
	}

	if backupCount > maxBackups {
		t.Errorf("expected at most %d backup files, got %d", maxBackups, backupCount)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Level != "INFO" {
		t.Errorf("default level should be INFO, got %s", cfg.Level)
	}
	if cfg.MaxSize != 100*1024*1024 {
		t.Errorf("default max size should be 100MB, got %d", cfg.MaxSize)
	}
	if cfg.MaxBackups != 5 {
		t.Errorf("default max backups should be 5, got %d", cfg.MaxBackups)
	}
	if cfg.MaxAge != 30 {
		t.Errorf("default max age should be 30, got %d", cfg.MaxAge)
	}
}

func TestNewFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		Level:        "DEBUG",
		Module:       "test",
		Colorize:     false,
		FileOutput:   true,
		FilePath:     filepath.Join(tmpDir, "config.log"),
		MaxSize:      1024,
		MaxBackups:   2,
		MaxAge:       7,
		RotateByDate: false,
	}

	l := NewFromConfig(cfg)
	l.Info("config test message")
	l.Sync()
	l.Close()

	content, err := os.ReadFile(filepath.Join(tmpDir, "config.log"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "config test message") {
		t.Errorf("file should contain log message, got: %s", string(content))
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	Init(
		WithLevel(DEBUG),
		WithWriters(&buf),
		WithColorize(false),
	)

	Info("global info")
	Debug("global debug")
	Warn("global warn")
	Error("global error")
	Sync()

	output := buf.String()
	if !strings.Contains(output, "global info") {
		t.Error("global Info should work")
	}
	if !strings.Contains(output, "global debug") {
		t.Error("global Debug should work")
	}
	if !strings.Contains(output, "global warn") {
		t.Error("global Warn should work")
	}
	if !strings.Contains(output, "global error") {
		t.Error("global Error should work")
	}
}

func TestGlobalLoggerFormat(t *testing.T) {
	var buf bytes.Buffer
	Init(
		WithLevel(DEBUG),
		WithWriters(&buf),
		WithColorize(false),
	)

	Infof("formatted %s %d", "test", 42)
	Sync()

	output := buf.String()
	if !strings.Contains(output, "formatted test 42") {
		t.Errorf("global Infof should format message, got: %s", output)
	}
}

func TestGlobalSetLevel(t *testing.T) {
	var buf bytes.Buffer
	Init(
		WithLevel(DEBUG),
		WithWriters(&buf),
		WithColorize(false),
	)

	SetLevel(ERROR)
	if GetLevel() != ERROR {
		t.Error("global GetLevel should return ERROR")
	}

	Info("should not appear")
	Error("should appear")
	Sync()

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("INFO should be filtered after SetLevel(ERROR)")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("ERROR should pass through")
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				l.Info("concurrent log", "goroutine", id, "iteration", j)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	l.Sync()

	output := buf.String()
	lines := strings.Count(output, "\n")
	if lines < 900 {
		t.Errorf("expected at least 900 log lines, got %d", lines)
	}
}

func TestLogEntryContainsCallerInfo(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	defer l.Close()

	l.Info("caller test")
	l.Sync()

	output := buf.String()

	fileLinePattern := regexp.MustCompile(`logger_test\.go:\d+`)
	if !fileLinePattern.MatchString(output) {
		t.Errorf("log should contain file:line from test file, got: %s", output)
	}

	if !strings.Contains(output, "TestLogEntryContainsCallerInfo") {
		t.Errorf("log should contain function name, got: %s", output)
	}
}
