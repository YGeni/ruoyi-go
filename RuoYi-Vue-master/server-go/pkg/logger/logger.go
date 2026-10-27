// Package logger 基于 zap 提供全局日志。
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log 全局 Logger，Init 后可用。
	Log   *zap.Logger
	Sugar *zap.SugaredLogger
)

// Init 初始化全局日志。
func Init(level string) error {
	lv, err := zapcore.ParseLevel(level)
	if err != nil {
		lv = zapcore.DebugLevel
	}
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(lv)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = l
	Sugar = l.Sugar()
	return nil
}

// Sync 刷新缓冲，进程退出前调用。
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
