package logs

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfg = zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "lvl",
		NameKey:        "mod",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core = zapcore.NewTee(zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevelAt(zapcore.InfoLevel)))
	logger = zap.New(core).Sugar()
)

func Desugar() *zap.Logger { return logger.Desugar() }

func With(args ...interface{}) *zap.SugaredLogger     { return logger.With(args...) }
func Sync() error                                     { return logger.Sync() }
func Debug(args ...interface{})                       { logger.Debug(args...) }
func Info(args ...interface{})                        { logger.Info(args...) }
func Warn(args ...interface{})                        { logger.Warn(args...) }
func Error(args ...interface{})                       { logger.Error(args...) }
func Fatal(args ...interface{})                       { logger.Fatal(args...) }
func Debugf(template string, args ...interface{})     { logger.Debugf(template, args...) }
func Infof(template string, args ...interface{})      { logger.Infof(template, args...) }
func Warnf(template string, args ...interface{})      { logger.Warnf(template, args...) }
func Errorf(template string, args ...interface{})     { logger.Errorf(template, args...) }
func Fatalf(template string, args ...interface{})     { logger.Fatalf(template, args...) }
func Debugw(msg string, keysAndValues ...interface{}) { logger.Debugw(msg, keysAndValues...) }
func Infow(msg string, keysAndValues ...interface{})  { logger.Infow(msg, keysAndValues...) }
func Warnw(msg string, keysAndValues ...interface{})  { logger.Warnw(msg, keysAndValues...) }
func Errorw(msg string, keysAndValues ...interface{}) { logger.Errorw(msg, keysAndValues...) }
func Fatalw(msg string, keysAndValues ...interface{}) { logger.Fatalw(msg, keysAndValues...) }
