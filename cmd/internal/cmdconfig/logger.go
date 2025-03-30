package cmdconfig

import (
	"github.com/tetsuo/fortune/internal/wraperr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(level zapcore.Level, devMode bool) (log *zap.Logger, err error) {
	defer wraperr.Wrap(&err, "NewLogger(%q, %v)", level.String(), devMode)

	loggerEncodingConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "severity",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    loggerEncodeLevel(),
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.NanosDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	loggerCfg := &zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Encoding:         "json",
		EncoderConfig:    loggerEncodingConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		Sampling:         nil,
	}
	if devMode {
		loggerCfg.Encoding = "console"
		loggerCfg.Development = true
		loggerCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	log, err = loggerCfg.Build(zap.AddStacktrace(zap.DPanicLevel))
	if err == nil {
		zap.ReplaceGlobals(log)
	}
	return
}

func loggerEncodeLevel() zapcore.LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		switch l {
		case zapcore.DebugLevel:
			enc.AppendString("DEBUG")
		case zapcore.InfoLevel:
			enc.AppendString("INFO")
		case zapcore.WarnLevel:
			enc.AppendString("WARNING")
		case zapcore.ErrorLevel:
			enc.AppendString("ERROR")
		case zapcore.DPanicLevel:
			enc.AppendString("CRITICAL")
		case zapcore.PanicLevel:
			enc.AppendString("ALERT")
		case zapcore.FatalLevel:
			enc.AppendString("EMERGENCY")
		}
	}
}
