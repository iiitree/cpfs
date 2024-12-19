package meta

import (
	"context"
	"os"
	"time"
)

// Storage 定义存储接口
type Storage interface {
	// Save 保存数据
	Save(ctx context.Context, key string, data []byte) error
	// Load 加载数据
	Load(ctx context.Context, key string) ([]byte, error)
	// Delete 删除数据
	Delete(ctx context.Context, key string) error
	// List 列出指定前缀的所有键
	List(ctx context.Context, prefix string) ([]string, error)
	// Sync 同步数据到持久化存储
	Sync() error
}

// StorageConfig 存储配置
type StorageConfig struct {
	// 存储根目录
	RootDir string
	// 同步间隔
	SyncInterval time.Duration
	// 文件权限
	FileMode os.FileMode
	// 是否启用压缩
	EnableCompression bool
}

// DefaultStorageConfig 返回默认配置
func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		RootDir:           "data/meta",
		SyncInterval:      time.Second * 5,
		FileMode:          0644,
		EnableCompression: false,
	}
}
