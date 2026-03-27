package logger

import (
	"os"
	"path"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Init 初始化日志系统（写入文件 + 控制台）
func Init() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		logrus.WithError(err).Error("创建日志目录失败")
		return
	}

	writer, err := rotatelogs.New(
		path.Join("logs", "%Y-%m-%d.log"),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		logrus.WithError(err).Error("初始化日志轮转失败")
		return
	}

	formatter := &logrus.TextFormatter{
		FullTimestamp:    true,
		PadLevelText:     true,
		QuoteEmptyFields: true,
	}

	log.SetOutput(writer)
	log.SetFormatter(formatter)

	// 同时输出到控制台
	log.AddHook(lfshook.NewHook(
		lfshook.WriterMap{
			logrus.DebugLevel: os.Stdout,
			logrus.InfoLevel:  os.Stdout,
			logrus.WarnLevel:  os.Stderr,
			logrus.ErrorLevel: os.Stderr,
			logrus.FatalLevel: os.Stderr,
			logrus.PanicLevel: os.Stderr,
		},
		formatter,
	))
}

// Sync 同步日志（logrus 无需显式同步，保留兼容接口）
func Sync() {}

func WithField(key string, value interface{}) *logrus.Entry { return log.WithField(key, value) }
func WithFields(fields logrus.Fields) *logrus.Entry         { return log.WithFields(fields) }
func Debug(args ...interface{})                             { log.Debug(args...) }
func Debugf(format string, args ...interface{})             { log.Debugf(format, args...) }
func Info(args ...interface{})                              { log.Info(args...) }
func Infof(format string, args ...interface{})              { log.Infof(format, args...) }
func Warn(args ...interface{})                              { log.Warn(args...) }
func Warnf(format string, args ...interface{})              { log.Warnf(format, args...) }
func Error(args ...interface{})                             { log.Error(args...) }
func Errorf(format string, args ...interface{})             { log.Errorf(format, args...) }
func Fatal(args ...interface{})                             { log.Fatal(args...) }
func Fatalf(format string, args ...interface{})             { log.Fatalf(format, args...) }
func Panic(args ...interface{})                             { log.Panic(args...) }
func Panicf(format string, args ...interface{})             { log.Panicf(format, args...) }

// SetLevel 动态调整日志级别（debug/info/warn/error）
func SetLevel(level string) {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		log.Warnf("不支持的日志级别: %s", level)
		return
	}
	log.SetLevel(l)
}
