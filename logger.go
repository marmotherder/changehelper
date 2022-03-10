package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sLogger *zap.SugaredLogger

func setupLogger(level int) error {
	loggerCfg := zap.NewDevelopmentConfig()
	switch level {
	case 0:
		loggerCfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	case 1:
		loggerCfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case 2:
		loggerCfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case 3:
		loggerCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	default:
		loggerCfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	logger, err := loggerCfg.Build()
	if err != nil {
		return err
	}
	sLogger = logger.Sugar()
	return nil
}
