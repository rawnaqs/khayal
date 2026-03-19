package log

import (
	"context"
	"log/slog"
	"os"
)

type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &multiHandler{handlers: cloneHandlers(h.handlers, func(h slog.Handler) slog.Handler {
		return h.WithAttrs(attrs)
	})}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	return &multiHandler{handlers: cloneHandlers(h.handlers, func(h slog.Handler) slog.Handler {
		return h.WithGroup(name)
	})}
}

func cloneHandlers(handlers []slog.Handler, fn func(slog.Handler) slog.Handler) []slog.Handler {
	result := make([]slog.Handler, len(handlers))
	for i, h := range handlers {
		result[i] = fn(h)
	}
	return result
}

func NewMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type LoggerSetup struct {
	MainLogger   *slog.Logger
	WorkerLogger *slog.Logger
	file         *RotatingLogFile
}

func SetupLogger(logFile string, configPath string, maxSizeMB, maxFiles int, level, workerLevel string) (*LoggerSetup, error) {
	rotatingFile, err := NewRotatingLogFile(logFile, configPath, maxSizeMB, maxFiles)
	if err != nil {
		return nil, err
	}

	mainLvl := parseLevel(level)
	workerLvl := parseLevel(workerLevel)
	if workerLevel == "" {
		workerLvl = mainLvl
	}

	fileHandler := slog.NewJSONHandler(rotatingFile, &slog.HandlerOptions{
		Level: mainLvl,
	})
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: mainLvl,
	})

	mainHandler := NewMultiHandler(fileHandler, stdoutHandler)
	mainLogger := slog.New(mainHandler)

	workerFileHandler := slog.NewJSONHandler(rotatingFile, &slog.HandlerOptions{
		Level: workerLvl,
	})
	workerStdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: workerLvl,
	})

	workerHandler := NewMultiHandler(workerFileHandler, workerStdoutHandler)
	workerLogger := slog.New(workerHandler)

	return &LoggerSetup{
		MainLogger:   mainLogger,
		WorkerLogger: workerLogger,
		file:         rotatingFile,
	}, nil
}

func (ls *LoggerSetup) Close() error {
	if ls.file != nil {
		return ls.file.Close()
	}
	return nil
}

func (ls *LoggerSetup) Sync() error {
	if ls.file != nil {
		return ls.file.Sync()
	}
	return nil
}

func (ls *LoggerSetup) RotatingFile() *RotatingLogFile {
	return ls.file
}
