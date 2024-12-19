package meta

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cpfs/internal/logger"

	"go.uber.org/zap"
)

// FileStorage 实现基于文件的存储
type FileStorage struct {
	config *StorageConfig
	mu     sync.RWMutex
	cache  map[string][]byte
	dirty  map[string]bool
	stopCh chan struct{}
}

// NewFileStorage 创建新的文件存储实例
func NewFileStorage(config *StorageConfig) (*FileStorage, error) {
	if config == nil {
		config = DefaultStorageConfig()
	}

	// 创建存储目录
	if err := os.MkdirAll(config.RootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	fs := &FileStorage{
		config: config,
		cache:  make(map[string][]byte),
		dirty:  make(map[string]bool),
		stopCh: make(chan struct{}),
	}

	// 加载现有文件到缓存
	if err := fs.loadExistingFiles(); err != nil {
		return nil, fmt.Errorf("failed to load existing files: %v", err)
	}

	// 启动后台同步
	go fs.syncLoop()

	return fs, nil
}

// loadExistingFiles 加载现有文件到缓存
func (fs *FileStorage) loadExistingFiles() error {
	return filepath.Walk(fs.config.RootDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// 将文件路径转换为键
			key := fs.pathToKey(filePath)

			// 读取文件内容
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %v", filePath, err)
			}

			// 添加到缓存
			fs.cache[key] = data
		}
		return nil
	})
}

// validatePath 验证路径是否合法
func (fs *FileStorage) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path is not allowed")
	}

	// 检查路径是否包含无效字符
	if strings.ContainsAny(path, "\x00") {
		return fmt.Errorf("path contains invalid characters")
	}

	// 确保路径在存储根目录下
	fullPath := filepath.Clean(filepath.Join(fs.config.RootDir, path))
	if !strings.HasPrefix(fullPath, fs.config.RootDir) {
		return fmt.Errorf("path escapes root directory")
	}

	return nil
}

// pathToKey 将文件路径转换为键
func (fs *FileStorage) pathToKey(path string) string {
	// 移除根目录前缀
	key := strings.TrimPrefix(path, fs.config.RootDir)
	// 移除开头的路径分隔符
	key = strings.TrimPrefix(key, string(filepath.Separator))
	// 将路径分隔符转换为统一格式
	key = "/" + strings.ReplaceAll(key, string(filepath.Separator), "/")
	return key
}

// keyToPath 将键转换为文件路径
func (fs *FileStorage) keyToPath(key string) (string, error) {
	// 规范化键
	key = strings.TrimPrefix(key, "/")

	// 验证路径
	if err := fs.validatePath(key); err != nil {
		return "", err
	}

	// 转换为系统路径
	return filepath.Join(fs.config.RootDir, key), nil
}

// Save 保存数据
func (fs *FileStorage) Save(ctx context.Context, key string, data []byte) error {
	if key == "" {
		return fmt.Errorf("empty key is not allowed")
	}

	// 规范化key
	key = normalizePath(key)
	if key == "/" {
		return fmt.Errorf("root path is not allowed as key")
	}

	// 验证路径合法性
	if _, err := fs.keyToPath(strings.TrimPrefix(key, "/")); err != nil {
		return fmt.Errorf("invalid key: %v", err)
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 如果启用了压缩，压缩数据
	if fs.config.EnableCompression {
		compressedData, err := fs.CompressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress data: %v", err)
		}
		data = compressedData
	}

	// 更新缓存
	fs.cache[key] = data
	fs.dirty[key] = true

	logger.Info("Saved data to storage",
		zap.String("key", key),
		zap.Int("size", len(data)),
	)

	return nil
}

// Load 加载数据
func (fs *FileStorage) Load(ctx context.Context, key string) ([]byte, error) {
	// 规范化key
	key = normalizePath(key)

	fs.mu.RLock()
	// 检查缓存
	if data, ok := fs.cache[key]; ok {
		fs.mu.RUnlock()
		return data, nil
	}
	fs.mu.RUnlock()

	// 验证并获取文件路径
	path, err := fs.keyToPath(strings.TrimPrefix(key, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid key: %v", err)
	}

	// 从文件加载
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, err
	}

	// 如果启用了压缩，解压数据
	if fs.config.EnableCompression {
		data, err = fs.DecompressData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %v", err)
		}
	}

	// 更新缓存
	fs.mu.Lock()
	fs.cache[key] = data
	fs.mu.Unlock()

	return data, nil
}

// Delete 删除数据
func (fs *FileStorage) Delete(ctx context.Context, key string) error {
	// 规范化key
	key = normalizePath(key)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 从缓存中删除
	delete(fs.cache, key)
	fs.dirty[key] = true

	// 从文件系统删除
	path, err := fs.keyToPath(strings.TrimPrefix(key, "/"))
	if err != nil {
		return fmt.Errorf("invalid key: %v", err)
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	logger.Info("Deleted data from storage",
		zap.String("key", key),
	)

	return nil
}

// List 列出指定前缀的所有键
func (fs *FileStorage) List(ctx context.Context, prefix string) ([]string, error) {
	prefix = normalizePath(prefix)

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var keys []string
	for key := range fs.cache {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Sync 同步数据到磁盘
func (fs *FileStorage) Sync() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for key, data := range fs.cache {
		if fs.dirty[key] {
			// 获取文件路径
			path, err := fs.keyToPath(strings.TrimPrefix(key, "/"))
			if err != nil {
				return fmt.Errorf("invalid key while syncing: %v", err)
			}

			// 创建目录
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", dir, err)
			}

			// 写入文件
			if err := os.WriteFile(path, data, fs.config.FileMode); err != nil {
				return fmt.Errorf("failed to write file %s: %v", path, err)
			}

			delete(fs.dirty, key)
		}
	}

	return nil
}

// syncLoop 后台同步循环
func (fs *FileStorage) syncLoop() {
	ticker := time.NewTicker(fs.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fs.Sync(); err != nil {
				logger.Error("Failed to sync storage",
					zap.Error(err),
				)
			}
		case <-fs.stopCh:
			// 最后执行一次同步
			if err := fs.Sync(); err != nil {
				logger.Error("Failed to sync storage during shutdown",
					zap.Error(err),
				)
			}
			return
		}
	}
}

// Close 关闭存储
func (fs *FileStorage) Close() error {
	close(fs.stopCh)
	return fs.Sync()
}

// CompressData 压缩数据
func (fs *FileStorage) CompressData(data []byte) ([]byte, error) {
	if !fs.config.EnableCompression {
		return data, nil
	}
	// TODO: 实现数据压缩
	return data, nil
}

// DecompressData 解压数据
func (fs *FileStorage) DecompressData(data []byte) ([]byte, error) {
	if !fs.config.EnableCompression {
		return data, nil
	}
	// TODO: 实现数据解压
	return data, nil
}
