package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoggerInitialization(t *testing.T) {
	// 初始化日志系统
	err := InitLogger(true)
	assert.NoError(t, err)
	assert.NotNil(t, Log)

	// 测试日志文件是否创建
	_, err = os.Stat(logFile)
	assert.NoError(t, err)

	// 测试各个日志级别
	Debug("Debug message", zap.String("test", "debug"))
	Info("Info message", zap.String("test", "info"))
	Warn("Warn message", zap.String("test", "warn"))
	Error("Error message", zap.String("test", "error"))

	// 测试同步
	err = Sync()
	assert.NoError(t, err)
}

func TestLoggerWithFields(t *testing.T) {
	err := InitLogger(true)
	assert.NoError(t, err)

	// 测试带字段的日志
	Info("Test message with fields",
		zap.String("string_field", "test"),
		zap.Int("int_field", 123),
		zap.Bool("bool_field", true),
	)

	// 验证日志文件存在
	_, err = os.Stat(logFile)
	assert.NoError(t, err)
}

func TestMain(m *testing.M) {
	// 测试前清理
	_ = os.RemoveAll("logs")

	// 运行测试
	code := m.Run()

	// 测试后清理
	_ = os.RemoveAll("logs")

	os.Exit(code)
}
