package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Configure initializes and configures a zap logger.
func Setup() (*zap.Logger, error) {
	// Default level is info
	logLevel := zapcore.InfoLevel

	// Read the log level from an environment variable
	envLogLevel := os.Getenv("LOG_LEVEL")
	if envLogLevel != "" {
		err := logLevel.Set(strings.ToLower(envLogLevel))
		if err != nil {
			return nil, err
		}
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)

	return config.Build()
}
