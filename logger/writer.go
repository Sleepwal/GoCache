package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type RotatingFileWriter struct {
	mu          sync.Mutex
	file        *os.File
	filePath    string
	dir         string
	baseName    string
	ext         string
	maxSize     int64
	maxBackups  int
	maxAge      int
	rotateByDate bool
	currentSize int64
	currentDate string
}

func NewRotatingFileWriter(filePath string, maxSize int64, maxBackups, maxAge int, rotateByDate bool) *RotatingFileWriter {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	w := &RotatingFileWriter{
		filePath:    filePath,
		dir:         dir,
		baseName:    name,
		ext:         ext,
		maxSize:     maxSize,
		maxBackups:  maxBackups,
		maxAge:      maxAge,
		rotateByDate: rotateByDate,
		currentDate: time.Now().Format("2006-01-02"),
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to create log directory %s: %v\n", dir, err)
		return w
	}

	w.openFile()

	return w
}

func (w *RotatingFileWriter) openFile() {
	f, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to open log file %s: %v\n", w.filePath, err)
		return
	}

	w.file = f
	info, err := f.Stat()
	if err == nil {
		w.currentSize = info.Size()
	} else {
		w.currentSize = 0
	}
}

func (w *RotatingFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		w.openFile()
		if w.file == nil {
			return 0, fmt.Errorf("logger: log file not available")
		}
	}

	if w.rotateByDate {
		today := time.Now().Format("2006-01-02")
		if today != w.currentDate {
			w.rotateByDateInternal(today)
			w.currentDate = today
		}
	}

	if w.maxSize > 0 && w.currentSize+int64(len(p)) > w.maxSize {
		w.rotateBySize()
	}

	n, err = w.file.Write(p)
	w.currentSize += int64(n)
	return n, err
}

func (w *RotatingFileWriter) rotateBySize() {
	w.file.Close()

	backupName := fmt.Sprintf("%s-%s%s",
		w.baseName,
		time.Now().Format("20060102-150405"),
		w.ext,
	)
	backupPath := filepath.Join(w.dir, backupName)

	if err := os.Rename(w.filePath, backupPath); err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to rotate log file: %v\n", err)
		w.openFile()
		return
	}

	w.openFile()
	w.cleanOldBackups()
}

func (w *RotatingFileWriter) rotateByDateInternal(newDate string) {
	w.file.Close()

	backupName := fmt.Sprintf("%s-%s%s",
		w.baseName,
		w.currentDate,
		w.ext,
	)
	backupPath := filepath.Join(w.dir, backupName)

	if _, err := os.Stat(backupPath); err == nil {
		backupName = fmt.Sprintf("%s-%s-%d%s",
			w.baseName,
			w.currentDate,
			time.Now().UnixNano(),
			w.ext,
		)
		backupPath = filepath.Join(w.dir, backupName)
	}

	if err := os.Rename(w.filePath, backupPath); err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to rotate log file by date: %v\n", err)
		w.openFile()
		return
	}

	w.openFile()
	w.cleanOldBackups()
}

func (w *RotatingFileWriter) cleanOldBackups() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}

	var backups []backupInfo
	prefix := w.baseName + "-"

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, w.ext) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backupInfo{
			name:    name,
			modTime: info.ModTime(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	now := time.Now()
	for i, b := range backups {
		shouldDelete := false

		if w.maxBackups > 0 && i >= w.maxBackups {
			shouldDelete = true
		}

		if w.maxAge > 0 && now.Sub(b.modTime).Hours() > float64(w.maxAge*24) {
			shouldDelete = true
		}

		if shouldDelete {
			os.Remove(filepath.Join(w.dir, b.name))
		}
	}
}

type backupInfo struct {
	name    string
	modTime time.Time
}

func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
