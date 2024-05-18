package logger

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}
	zap.ReplaceGlobals(logger)
}

func Info(args ...interface{}) {
	zap.S().Info(args...)
}

func Infof(template string, args ...interface{}) {
	zap.S().Infof(template, args...)
}

func Debug(args ...interface{}) {
	zap.S().Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	zap.S().Debugf(template, args...)
}

func Error(args ...interface{}) {
	zap.S().Error(args...)
}

func Errorf(template string, args ...interface{}) {
	zap.S().Errorf(template, args...)
}

func Fatal(args ...interface{}) {
	zap.S().Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	zap.S().Fatalf(template, args...)
}
