package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Init configures global slog logger.
func Init() error {
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf(
		"log_%s.json",
		time.Now().Format("2006-01-02_15-04-05"),
	)

	path := filepath.Join("logs", filename)

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		return err
	}

	writer := io.MultiWriter(os.Stdout, file)

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("logger initialized")

	return nil
}
