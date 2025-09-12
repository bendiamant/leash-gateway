package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config represents logger configuration
type Config struct {
	Level       string
	Format      string
	Output      string
	AddSource   bool
	Development bool
}

// NewLogger creates a new structured logger
func NewLogger(config Config) (*zap.Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}

	// Create encoder config
	var encoderConfig zapcore.EncoderConfig
	if config.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
	}

	// Configure time encoding
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Configure level encoding
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	// Configure caller encoding
	if config.AddSource {
		encoderConfig.CallerKey = "caller"
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	// Create encoder
	var encoder zapcore.Encoder
	switch config.Format {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "text", "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(zapcore.Lock(zapcore.AddSync(getWriter(config.Output)))),
		level,
	)

	// Create logger
	var options []zap.Option
	if config.AddSource {
		options = append(options, zap.AddCaller())
	}
	if config.Development {
		options = append(options, zap.Development())
	}
	options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))

	logger := zap.New(core, options...)

	return logger, nil
}

// getWriter returns the appropriate writer for the given output
func getWriter(output string) zapcore.WriteSyncer {
	switch output {
	case "stdout":
		return zapcore.AddSync(os.Stdout)
	case "stderr":
		return zapcore.AddSync(os.Stderr)
	default:
		// Default to stdout for unsupported outputs
		return zapcore.AddSync(os.Stdout)
	}
}
