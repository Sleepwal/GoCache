package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

type logEntry struct {
	Timestamp time.Time
	Level     Level
	File      string
	Line      int
	Function  string
	Message   string
	Fields    []any
	Err       error
}

type flushRequest struct {
	done chan struct{}
}

type Logger struct {
	mu       sync.RWMutex
	level    Level
	writers  []io.Writer
	module   string
	fields   []any
	quit     chan struct{}
	entryCh  chan *logEntry
	flushCh  chan *flushRequest
	wg       sync.WaitGroup
	pool     sync.Pool
	running  bool
	colorize bool
	shared   bool
}

func New(opts ...Option) *Logger {
	l := &Logger{
		level:    INFO,
		writers:  []io.Writer{os.Stdout},
		module:   "",
		quit:     make(chan struct{}),
		entryCh:  make(chan *logEntry, 4096),
		flushCh:  make(chan *flushRequest, 16),
		colorize: true,
		shared:   false,
		pool: sync.Pool{
			New: func() any {
				return &logEntry{}
			},
		},
	}

	for _, opt := range opts {
		opt(l)
	}

	l.running = true
	l.wg.Add(1)
	go l.processLoop()

	return l
}

func (l *Logger) processLoop() {
	defer l.wg.Done()
	for {
		select {
		case entry := <-l.entryCh:
			l.writeEntry(entry)
			l.pool.Put(entry)
		case req := <-l.flushCh:
			l.drainEntries()
			close(req.done)
		case <-l.quit:
			l.drainEntries()
			return
		}
	}
}

func (l *Logger) drainEntries() {
	for {
		select {
		case entry := <-l.entryCh:
			l.writeEntry(entry)
			l.pool.Put(entry)
		default:
			return
		}
	}
}

func (l *Logger) writeEntry(entry *logEntry) {
	l.mu.RLock()
	writers := l.writers
	colorize := l.colorize
	l.mu.RUnlock()

	plainLine := l.formatEntry(entry, false)

	for _, w := range writers {
		if colorize && (w == os.Stdout || w == os.Stderr) {
			fmt.Fprintln(w, l.formatEntry(entry, true))
		} else {
			fmt.Fprintln(w, plainLine)
		}
	}

	if entry.Level == FATAL {
		os.Exit(1)
	}
}

func (l *Logger) formatEntry(entry *logEntry, color bool) string {
	ts := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	levelStr := entry.Level.String()

	var levelFormatted string
	if color {
		levelFormatted = colorizeLevel(entry.Level, levelStr)
	} else {
		levelFormatted = levelStr
	}

	caller := fmt.Sprintf("%s:%d %s", entry.File, entry.Line, entry.Function)

	msg := entry.Message
	if entry.Err != nil {
		msg = fmt.Sprintf("%s error=%v", entry.Message, entry.Err)
	}

	if len(entry.Fields) > 0 {
		fields := formatFields(entry.Fields)
		return fmt.Sprintf("%s [%s] [%s] %s %s", ts, levelFormatted, caller, msg, fields)
	}

	return fmt.Sprintf("%s [%s] [%s] %s", ts, levelFormatted, caller, msg)
}

func formatFields(fields []any) string {
	if len(fields) == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(fields); i += 2 {
		if i > 0 {
			b.WriteByte(' ')
		}
		if i+1 < len(fields) {
			fmt.Fprintf(&b, "%v=%v", fields[i], fields[i+1])
		} else {
			fmt.Fprintf(&b, "%v=<missing>", fields[i])
		}
	}
	return b.String()
}

func colorizeLevel(level Level, s string) string {
	const (
		reset  = "\033[0m"
		gray   = "\033[37m"
		green  = "\033[32m"
		yellow = "\033[33m"
		red    = "\033[31m"
		bold   = "\033[1m"
	)
	switch level {
	case DEBUG:
		return gray + s + reset
	case INFO:
		return green + s + reset
	case WARN:
		return yellow + s + reset
	case ERROR:
		return red + s + reset
	case FATAL:
		return bold + red + s + reset
	default:
		return s
	}
}

func (l *Logger) log(level Level, callerSkip int, msg string, fields []any) {
	if level < l.getLevel() {
		return
	}

	entry := l.pool.Get().(*logEntry)
	entry.Timestamp = time.Now()
	entry.Level = level
	entry.Message = msg
	entry.Err = nil
	entry.Fields = l.mergeFields(fields)

	pc, file, line, ok := runtime.Caller(callerSkip)
	if ok {
		entry.File = shortFile(file)
		entry.Line = line
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			entry.Function = shortFuncName(fn.Name())
		} else {
			entry.Function = "???"
		}
	} else {
		entry.File = "???"
		entry.Line = 0
		entry.Function = "???"
	}

	select {
	case l.entryCh <- entry:
	default:
		l.writeEntry(entry)
		l.pool.Put(entry)
	}
}

func (l *Logger) logError(level Level, callerSkip int, msg string, err error, fields []any) {
	if level < l.getLevel() {
		return
	}

	entry := l.pool.Get().(*logEntry)
	entry.Timestamp = time.Now()
	entry.Level = level
	entry.Message = msg
	entry.Err = err
	entry.Fields = l.mergeFields(fields)

	pc, file, line, ok := runtime.Caller(callerSkip)
	if ok {
		entry.File = shortFile(file)
		entry.Line = line
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			entry.Function = shortFuncName(fn.Name())
		} else {
			entry.Function = "???"
		}
	} else {
		entry.File = "???"
		entry.Line = 0
		entry.Function = "???"
	}

	select {
	case l.entryCh <- entry:
	default:
		l.writeEntry(entry)
		l.pool.Put(entry)
	}
}

func (l *Logger) mergeFields(extra []any) []any {
	l.mu.RLock()
	base := l.fields
	l.mu.RUnlock()

	if len(base) == 0 && l.module == "" {
		return extra
	}

	merged := make([]any, 0, len(base)+len(extra)+2)
	if l.module != "" {
		merged = append(merged, "module", l.module)
	}
	merged = append(merged, base...)
	merged = append(merged, extra...)
	return merged
}

func (l *Logger) getLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

func shortFile(path string) string {
	sepCount := 0
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			sepCount++
			if sepCount == 2 {
				return path[i+1:]
			}
		}
	}
	return path
}

func shortFuncName(name string) string {
	idx := strings.LastIndex(name, "/")
	if idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

func (l *Logger) Debug(msg string, fields ...any) {
	l.log(DEBUG, 2, msg, fields)
}

func (l *Logger) Info(msg string, fields ...any) {
	l.log(INFO, 2, msg, fields)
}

func (l *Logger) Warn(msg string, fields ...any) {
	l.log(WARN, 2, msg, fields)
}

func (l *Logger) Error(msg string, fields ...any) {
	l.log(ERROR, 2, msg, fields)
}

func (l *Logger) ErrorErr(msg string, err error, fields ...any) {
	l.logError(ERROR, 2, msg, err, fields)
}

func (l *Logger) Fatal(msg string, fields ...any) {
	l.log(FATAL, 2, msg, fields)
}

func (l *Logger) FatalErr(msg string, err error, fields ...any) {
	l.logError(FATAL, 2, msg, err, fields)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.log(DEBUG, 2, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Infof(format string, args ...any) {
	l.log(INFO, 2, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.log(WARN, 2, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.log(ERROR, 2, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.log(FATAL, 2, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) With(fields ...any) *Logger {
	l.mu.RLock()
	baseFields := make([]any, len(l.fields))
	copy(baseFields, l.fields)
	writers := make([]io.Writer, len(l.writers))
	copy(writers, l.writers)
	level := l.level
	module := l.module
	colorize := l.colorize
	l.mu.RUnlock()

	newLogger := &Logger{
		level:    level,
		writers:  writers,
		module:   module,
		fields:   append(baseFields, fields...),
		quit:     l.quit,
		entryCh:  l.entryCh,
		flushCh:  l.flushCh,
		colorize: colorize,
		pool:     l.pool,
		running:  l.running,
		shared:   true,
	}

	return newLogger
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) GetLevel() Level {
	return l.getLevel()
}

func (l *Logger) AddWriter(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writers = append(l.writers, w)
}

func (l *Logger) SetWriters(writers ...io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writers = writers
}

func (l *Logger) Close() error {
	l.mu.Lock()
	if !l.running || l.shared {
		l.mu.Unlock()
		return nil
	}
	l.running = false
	l.mu.Unlock()

	close(l.quit)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.writers {
		if closer, ok := w.(io.Closer); ok {
			closer.Close()
		}
	}
	l.writers = nil
	return nil
}

func (l *Logger) Sync() {
	req := &flushRequest{done: make(chan struct{})}
	select {
	case l.flushCh <- req:
		<-req.done
	default:
	}
}
