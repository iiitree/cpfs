package network

import (
	"context"
	"cpfs/internal/logger"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	// 初始化日志
	logger.InitLogger(true)
}

func TestGRPCServer(t *testing.T) {
	// 创建服务器
	opts := ServerOptions{
		Address:    "127.0.0.1:0",   // 使用随机端口
		MaxMsgSize: 4 * 1024 * 1024, // 4MB
	}

	server, err := NewGRPCServer(opts)
	assert.NoError(t, err)
	assert.NotNil(t, server)

	// 启动服务器
	go func() {
		err := server.Start()
		assert.NoError(t, err)
	}()

	// 等待服务器启动
	time.Sleep(time.Second)

	// 获取实际地址
	addr := server.GetAddress()
	assert.NotEmpty(t, addr)

	// 创建客户端连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	assert.NoError(t, err)
	defer conn.Close()

	// 停止服务器
	server.Stop()
}

func TestGRPCServerTLSFailure(t *testing.T) {
	opts := ServerOptions{
		Address:  "127.0.0.1:0",
		TLS:      true,
		CertFile: "non_existent.crt",
		KeyFile:  "non_existent.key",
	}

	_, err := NewGRPCServer(opts)
	assert.Error(t, err)
}
