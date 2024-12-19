package meta

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"cpfs/internal/logger"

	"go.uber.org/zap"
)

// MemoryStore 内存元数据存储实现
type MemoryStore struct {
	mu     sync.RWMutex
	data   map[string]*Metadata
	inodes uint64
	root   *Metadata
}

// NewMemoryStore 创建新的内存存储
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		data:   make(map[string]*Metadata),
		inodes: 0,
	}

	// 创建根目录
	root := &Metadata{
		Inode:      store.nextInode(),
		Name:       "/",
		Type:       TypeDirectory,
		Mode:       0755,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
		AccessTime: time.Now(),
		Version:    1,
	}

	store.root = root
	store.data["/"] = root

	return store
}

// normalizePath 标准化路径
func normalizePath(p string) string {
	// 替换所有反斜杠为正斜杠
	p = strings.ReplaceAll(p, "\\", "/")

	// 删除开头的 "./"
	p = strings.TrimPrefix(p, "./")

	// 确保路径以 / 开头
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// 使用 path.Clean 处理 . 和 .. 以及重复的斜杠
	p = path.Clean(p)

	// 确保返回至少是根目录
	if p == "." || p == "" {
		return "/"
	}

	return p
}

func (s *MemoryStore) nextInode() uint64 {
	s.inodes++
	return s.inodes
}

// Create 创建新文件
func (s *MemoryStore) Create(ctx context.Context, p string, mode os.FileMode) (*Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := normalizePath(p)

	// 检查父目录是否存在
	parent := path.Dir(filePath)
	parentMeta, exists := s.data[parent]
	if !exists {
		return nil, fmt.Errorf("parent directory not found: %s", parent)
	}

	// 确保父路径是目录
	if parentMeta.Type != TypeDirectory {
		return nil, fmt.Errorf("parent path is not a directory: %s", parent)
	}

	// 检查文件是否已存在
	if _, exists := s.data[filePath]; exists {
		return nil, fmt.Errorf("file already exists: %s", filePath)
	}

	// 创建新文件元数据
	now := time.Now()
	meta := &Metadata{
		Inode:      s.nextInode(),
		Name:       path.Base(filePath),
		Type:       TypeRegular,
		Size:       0,
		Mode:       mode,
		Links:      1,
		CreateTime: now,
		ModifyTime: now,
		AccessTime: now,
		Version:    1,
	}

	s.data[filePath] = meta
	logger.Info("Created new file",
		zap.String("path", filePath),
		zap.Uint64("inode", meta.Inode),
	)

	return meta, nil
}

// Get 获取文件元数据
func (s *MemoryStore) Get(ctx context.Context, p string) (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := normalizePath(p)
	meta, exists := s.data[filePath]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	meta.AccessTime = time.Now()
	return meta, nil
}

// Update 更新文件元数据
func (s *MemoryStore) Update(ctx context.Context, p string, meta *Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := normalizePath(p)
	if _, exists := s.data[filePath]; !exists {
		return fmt.Errorf("file not found: %s", filePath)
	}

	meta.ModifyTime = time.Now()
	meta.Version++
	s.data[filePath] = meta

	return nil
}

// Delete 删除文件
func (s *MemoryStore) Delete(ctx context.Context, p string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := normalizePath(p)
	if _, exists := s.data[filePath]; !exists {
		return fmt.Errorf("file not found: %s", filePath)
	}

	delete(s.data, filePath)
	return nil
}

// List 列出目录内容
func (s *MemoryStore) List(ctx context.Context, p string) ([]*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dirPath := normalizePath(p)

	// 检查目录是否存在
	dirMeta, exists := s.data[dirPath]
	if !exists {
		return nil, fmt.Errorf("directory not found: %s", dirPath)
	}

	// 确保是目录
	if dirMeta.Type != TypeDirectory {
		return nil, fmt.Errorf("path is not a directory: %s", dirPath)
	}

	var results []*Metadata
	for p, meta := range s.data {
		if path.Dir(p) == dirPath && p != dirPath {
			results = append(results, meta)
		}
	}

	return results, nil
}

// Mkdir 创建目录
func (s *MemoryStore) Mkdir(ctx context.Context, p string, mode os.FileMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dirPath := normalizePath(p)

	// 检查父目录
	parent := path.Dir(dirPath)
	parentMeta, exists := s.data[parent]
	if !exists {
		return fmt.Errorf("parent directory not found: %s", parent)
	}

	// 确保父路径是目录
	if parentMeta.Type != TypeDirectory {
		return fmt.Errorf("parent path is not a directory: %s", parent)
	}

	// 检查目录是否已存在
	if _, exists := s.data[dirPath]; exists {
		return fmt.Errorf("directory already exists: %s", dirPath)
	}

	// 创建目录元数据
	now := time.Now()
	meta := &Metadata{
		Inode:      s.nextInode(),
		Name:       path.Base(dirPath),
		Type:       TypeDirectory,
		Mode:       mode | os.ModeDir,
		Links:      1,
		CreateTime: now,
		ModifyTime: now,
		AccessTime: now,
		Version:    1,
	}

	s.data[dirPath] = meta
	return nil
}
