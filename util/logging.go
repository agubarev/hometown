package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DefaultLogger initializing default logger
func DefaultLogger(debug bool, logDir string) (*zap.Logger, error) {
	logDir = strings.TrimSpace(logDir)

	if logDir == "" {
		return nil, errors.New("empty log directory path")
	}

	// creating log directory if it doesn't exist
	if err := CreateDirectoryIfNotExists(logDir, 0777); err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// errors logfile
	//---------------------------------------------------------------------------
	errFilepath := filepath.Join(logDir, "errors.log")
	errFile, err := os.OpenFile(errFilepath, os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed to create error log file %s: %s", errFilepath, err)
	}
	errFileLog := zapcore.Lock(zapcore.AddSync(errFile))

	//---------------------------------------------------------------------------
	// regular logfile
	//---------------------------------------------------------------------------
	stdFilepath := filepath.Join(logDir, "standard.log")
	stdFile, err := os.OpenFile(stdFilepath, os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed to create standard log file %s: %s", errFilepath, err)
	}
	stdFileLog := zapcore.Lock(zapcore.AddSync(stdFile))

	//---------------------------------------------------------------------------
	// log enablers and conjunction
	//---------------------------------------------------------------------------
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	core := zapcore.NewTee(
		// files
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), errFileLog, highPriority),
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), stdFileLog, lowPriority),

		// stdout, stderr
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.Lock(zapcore.AddSync(os.Stderr)), highPriority),
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.Lock(zapcore.AddSync(os.Stdout)), lowPriority),
	)

	return zap.New(core), nil
}
