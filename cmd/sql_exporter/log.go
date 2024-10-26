package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/prometheus/common/promslog"
)

type logConfig struct {
	logger         *slog.Logger
	logFileHandler *os.File
}

// initLogFile opens the log file for writing if a log file is specified.
func initLogFile(logFile string) (*os.File, error) {
	if logFile == "" {
		return nil, nil
	}
	logFileHandler, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %w", err)
	}
	return logFileHandler, nil
}

// initLogConfig configures and initializes the logging system.
func initLogConfig(logLevel, logFormat string, logFile string) (*logConfig, error) {
	logFileHandler, err := initLogFile(logFile)
	if err != nil {
		return nil, err
	}

	if logFileHandler == nil {
		logFileHandler = os.Stderr
	}

	promslogConfig := &promslog.Config{
		Level:  &promslog.AllowedLevel{},
		Format: &promslog.AllowedFormat{},
		Style:  promslog.SlogStyle,
		Writer: logFileHandler,
	}

	if err := promslogConfig.Level.Set(logLevel); err != nil {
		return nil, err
	}

	if err := promslogConfig.Format.Set(logFormat); err != nil {
		return nil, err
	}
	// Initialize logger.
	logger := promslog.New(promslogConfig)

	return &logConfig{
		logger:         logger,
		logFileHandler: logFileHandler,
	}, nil
}
