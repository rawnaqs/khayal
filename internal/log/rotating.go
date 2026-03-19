package log

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
)

type RotatingLogFile struct {
	basePath    string
	maxSizeMB   int
	maxFiles    int
	currentFile *os.File
	currentSize int64
	mu          sync.Mutex
}

func NewRotatingLogFile(basePath string, configPath string, maxSizeMB, maxFiles int) (*RotatingLogFile, error) {
	basePath = config.MakeAbsolute(basePath, configPath)

	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	if maxFiles <= 0 {
		maxFiles = 5
	}

	rl := &RotatingLogFile{
		basePath:  basePath,
		maxSizeMB: maxSizeMB,
		maxFiles:  maxFiles,
	}

	if err := rl.openCurrentFile(); err != nil {
		return nil, err
	}

	return rl, nil
}

func (rl *RotatingLogFile) openCurrentFile() error {
	dir := filepath.Dir(rl.basePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(rl.basePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	rl.currentFile = f
	rl.currentSize = info.Size()

	return nil
}

func (rl *RotatingLogFile) Write(p []byte) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	maxSize := int64(rl.maxSizeMB) * 1024 * 1024

	if rl.currentSize+int64(len(p)) > maxSize {
		if err := rl.rotate(); err != nil {
			return 0, fmt.Errorf("failed to rotate: %w", err)
		}
	}

	n, err = rl.currentFile.Write(p)
	if err != nil {
		return n, err
	}

	rl.currentSize += int64(n)
	return n, nil
}

func (rl *RotatingLogFile) rotate() error {
	if err := rl.currentFile.Close(); err != nil {
		return fmt.Errorf("failed to close current file: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02-150405")
	rotatedPath := fmt.Sprintf("%s.%s", rl.basePath, timestamp)

	if err := os.Rename(rl.basePath, rotatedPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	if err := rl.compressFile(rotatedPath); err != nil {
		return fmt.Errorf("failed to compress rotated file: %w", err)
	}

	if err := rl.cleanupOldFiles(); err != nil {
		return fmt.Errorf("failed to cleanup old files: %w", err)
	}

	if err := rl.openCurrentFile(); err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	return nil
}

func (rl *RotatingLogFile) compressFile(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for compression: %w", err)
	}
	defer src.Close()

	dstPath := path + ".gz"
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer dst.Close()

	gz := gzip.NewWriter(dst)
	defer gz.Close()

	if _, err := io.Copy(gz, src); err != nil {
		os.Remove(dstPath)
		return fmt.Errorf("failed to compress: %w", err)
	}

	os.Remove(path)
	return nil
}

func (rl *RotatingLogFile) cleanupOldFiles() error {
	dir := filepath.Dir(rl.basePath)
	baseName := filepath.Base(rl.basePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	var logFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == baseName {
			continue
		}
		if strings.HasPrefix(name, baseName+".") {
			fullPath := filepath.Join(dir, name)
			logFiles = append(logFiles, fullPath)
		}
	}

	sort.Strings(logFiles)

	if len(logFiles) > rl.maxFiles {
		toDelete := logFiles[:len(logFiles)-rl.maxFiles]
		for _, f := range toDelete {
			os.Remove(f)
		}
	}

	return nil
}

func (rl *RotatingLogFile) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.currentFile != nil {
		return rl.currentFile.Close()
	}
	return nil
}

func (rl *RotatingLogFile) Sync() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.currentFile != nil {
		return rl.currentFile.Sync()
	}
	return nil
}
