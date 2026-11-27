// Package logger holds the controller's shared zap logger.
//
// Importing the package has no side effects: Logger discards everything until
// the program calls Init (cmd/main does). Tests and tools therefore never
// create log directories relative to wherever they happen to run.
package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is safe to use before Init; it is a no-op logger until then.
var Logger = zap.NewNop().Sugar()

// Init sends logs to dir/structured.log (JSON) and dir/readable.log (console
// format), creating dir if needed. An empty dir keeps the no-op logger.
func Init(dir string) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("logger: %w", err)
	}

	rotating := func(name string) zapcore.WriteSyncer {
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(dir, name),
			MaxSize:    100, // MB before rotation
			MaxBackups: 1,
			MaxAge:     30, // days
		})
	}

	jsonConfig := zap.NewProductionEncoderConfig()
	jsonConfig.TimeKey = "timestamp"
	jsonConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	humanConfig := zap.NewDevelopmentEncoderConfig()
	humanConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(jsonConfig), rotating("structured.log"), zapcore.DebugLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(humanConfig), rotating("readable.log"), zapcore.DebugLevel),
	)
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).Sugar()
	return nil
}
