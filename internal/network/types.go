package network

import (
	"context"
)

// ServerOptions 定义服务器选项
type ServerOptions struct {
	Address    string
	MaxMsgSize int
	TLS        bool
	CertFile   string
	KeyFile    string
}

// Server 定义网络服务器接口
type Server interface {
	// Start 启动服务器
	Start() error
	// Stop 停止服务器
	Stop()
	// GetAddress 获取服务器地址
	GetAddress() string
}

// Connection 定义网络连接接口
type Connection interface {
	// Send 发送数据
	Send(ctx context.Context, data []byte) error
	// Close 关闭连接
	Close() error
}
