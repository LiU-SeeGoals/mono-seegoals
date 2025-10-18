package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger *zap.SugaredLogger
)

func init() {
	// Log rotation for JSON logs (structured)
	jsonLogFile := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "../logs/structured.log", // Structured logs in JSON format
		MaxSize:    100,  // Max size in MB before rotation
		MaxBackups: 1,   // Max number of old log files to keep
		MaxAge:     30,  // Max number of days to retain old logs
		Compress:   false, // Compress old logs (gzip)
	})

	// Log rotation for human-readable logs
	humanLogFile := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "../logs/readable.log", // Readable logs for easy debugging
		MaxSize:    100,
		MaxBackups: 1,
		MaxAge:     30,
		Compress:   false,
	})

	// Create JSON encoder (structured logs)
	jsonEncoderConfig := zap.NewProductionEncoderConfig()
	jsonEncoderConfig.TimeKey = "timestamp"
	jsonEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // Human-readable timestamp
	jsonEncoder := zapcore.NewJSONEncoder(jsonEncoderConfig)

	// Create human-readable encoder (console-style logs)
	humanEncoderConfig := zap.NewDevelopmentEncoderConfig()
	humanEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	humanEncoder := zapcore.NewConsoleEncoder(humanEncoderConfig) // Pretty-print logs

	// Set up cores for logging
	jsonCore := zapcore.NewCore(jsonEncoder, jsonLogFile, zapcore.DebugLevel) // JSON logs
	humanCore := zapcore.NewCore(humanEncoder, humanLogFile, zapcore.DebugLevel) // Readable logs
	// consoleCore := zapcore.NewCore(humanEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel) // Console logs

	// Combine all cores
	logCore := zapcore.NewTee(jsonCore, humanCore)//, consoleCore)

	// Create the logger
	logger := zap.New(logCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Convert to SugaredLogger
	Logger = logger.Sugar()
}

/*
	// Example: Custom Log Level (e.g., "NOTICE")
	const NoticeLevel zapcore.Level = 1  // Between Debug (-1) and Info (0)
	
	// Define a custom level enabler
	func CustomLevelEnabler(level zapcore.Level) bool {
		return level == NoticeLevel || level >= zapcore.InfoLevel
	}

	// Example: Add custom level to the logger
	customCore := zapcore.NewCore(fileEncoder, logFile, zap.LevelEnablerFunc(CustomLevelEnabler))
	logCore := zapcore.NewTee(fileCore, consoleCore, customCore)

	// Log a message with the custom level (if supported in your logger setup)
	LoggerS.Desugar().Check(NoticeLevel, "This is a NOTICE level log").Write()
*/


