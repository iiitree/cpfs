package network

import (
	"cpfs/internal/logger"
	"net"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCServer 实现 gRPC 服务器
type GRPCServer struct {
	opts     ServerOptions
	server   *grpc.Server
	listener net.Listener
	mu       sync.Mutex
	running  bool
}

// NewGRPCServer 创建新的 gRPC 服务器
func NewGRPCServer(opts ServerOptions) (*GRPCServer, error) {
	var serverOpts []grpc.ServerOption

	// 设置消息大小限制
	if opts.MaxMsgSize > 0 {
		serverOpts = append(serverOpts,
			grpc.MaxRecvMsgSize(opts.MaxMsgSize),
			grpc.MaxSendMsgSize(opts.MaxMsgSize),
		)
	}

	// 配置 TLS
	if opts.TLS {
		creds, err := credentials.NewServerTLSFromFile(opts.CertFile, opts.KeyFile)
		if err != nil {
			return nil, err
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	// 创建 gRPC 服务器
	server := grpc.NewServer(serverOpts...)

	return &GRPCServer{
		opts:   opts,
		server: server,
	}, nil
}

// Start 启动服务器
func (s *GRPCServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	lis, err := net.Listen("tcp", s.opts.Address)
	if err != nil {
		return err
	}

	s.listener = lis
	s.running = true

	logger.Info("Starting gRPC server",
		zap.String("address", s.opts.Address),
		zap.Bool("tls", s.opts.TLS),
	)

	return s.server.Serve(lis)
}

// Stop 停止服务器
func (s *GRPCServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	logger.Info("Stopping gRPC server",
		zap.String("address", s.opts.Address))

	s.server.GracefulStop()
	s.running = false
}

// GetAddress 获取服务器地址
func (s *GRPCServer) GetAddress() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.opts.Address
}
