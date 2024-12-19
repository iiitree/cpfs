// pkg/meta/memory_store_test.go
package meta

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createParentDirs 递归创建父目录
func createParentDirs(store *MemoryStore, ctx context.Context, pathStr string) error {
	pathStr = normalizePath(pathStr)
	if pathStr == "/" {
		return nil
	}

	parent := path.Dir(pathStr)
	if parent != "/" {
		if err := createParentDirs(store, ctx, parent); err != nil {
			return err
		}
	}

	// 检查目录是否已存在
	if _, err := store.Get(ctx, pathStr); err == nil {
		return nil // 目录已存在
	}

	// 创建目录
	return store.Mkdir(ctx, pathStr, 0755)
}

func TestMemoryStoreCrossPlatform(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Windows style path",
			path:     "dir1\\subdir1\\file1.txt",
			expected: "/dir1/subdir1/file1.txt",
		},
		{
			name:     "Linux style path",
			path:     "dir2/subdir2/file2.txt",
			expected: "/dir2/subdir2/file2.txt",
		},
		{
			name:     "Mixed style path",
			path:     "dir3\\subdir3/file3.txt",
			expected: "/dir3/subdir3/file3.txt",
		},
		{
			name:     "Path with double slashes",
			path:     "dir4//subdir4//file4.txt",
			expected: "/dir4/subdir4/file4.txt",
		},
		{
			name:     "Path with dot",
			path:     "./dir5/./subdir5/file5.txt",
			expected: "/dir5/subdir5/file5.txt",
		},
		{
			name:     "Path with double dot",
			path:     "dir6/subdir6/../file6.txt",
			expected: "/dir6/file6.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建父目录
			parentPath := path.Dir(tt.expected)
			err := createParentDirs(store, ctx, parentPath)
			assert.NoError(t, err, "Failed to create parent directories for %s", tt.path)

			// 创建文件
			meta, err := store.Create(ctx, tt.path, 0644)
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", tt.path, err)
			}
			assert.NotNil(t, meta)

			// 验证文件
			getMeta, err := store.Get(ctx, tt.expected)
			if err != nil {
				t.Fatalf("Failed to get file %s: %v", tt.expected, err)
			}
			assert.Equal(t, meta.Inode, getMeta.Inode)

			// 验证目录内容
			dirMeta, err := store.List(ctx, path.Dir(tt.expected))
			if err != nil {
				t.Fatalf("Failed to list directory %s: %v", path.Dir(tt.expected), err)
			}

			// 验证文件在目录中
			found := false
			expectedName := path.Base(tt.expected)
			for _, m := range dirMeta {
				if m.Name == expectedName {
					found = true
					break
				}
			}
			assert.True(t, found, "File %s not found in directory listing", expectedName)
		})
	}
}

func TestMemoryStoreOSSpecific(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	tests := []struct {
		name string
		path string
	}{
		{"Simple path", "test1/path1"},
		{"Nested path", "test2/path2/subpath2"},
		{"Deep path", "test3/path3/subpath3/deep3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 分解路径并逐级创建目录
			parts := strings.Split(strings.Trim(tt.path, "/"), "/")
			currentPath := "/"

			for _, part := range parts {
				currentPath = path.Join(currentPath, part)
				// 尝试创建目录
				err := store.Mkdir(ctx, currentPath, 0755)
				// 如果目录已存在，继续处理
				if err != nil && !strings.Contains(err.Error(), "already exists") {
					t.Fatalf("Failed to create directory %s: %v", currentPath, err)
				}
			}

			// 验证使用不同的路径分隔符访问
			forwardPath := "/" + strings.Join(parts, "/")
			backPath := "/" + strings.Join(parts, "\\")

			// 获取并验证元数据
			meta1, err := store.Get(ctx, forwardPath)
			if err != nil {
				t.Fatalf("Failed to get with forward slashes (%s): %v", forwardPath, err)
			}
			assert.NotNil(t, meta1)

			meta2, err := store.Get(ctx, backPath)
			if err != nil {
				t.Fatalf("Failed to get with backslashes (%s): %v", backPath, err)
			}
			assert.NotNil(t, meta2)

			// 验证是否是同一个目录
			assert.Equal(t, meta1.Inode, meta2.Inode,
				"Different inodes for same path (%s vs %s)", forwardPath, backPath)

			// 验证目录类型
			assert.Equal(t, TypeDirectory, meta1.Type,
				"Path %s is not a directory", forwardPath)

			// 验证权限
			assert.Equal(t, os.FileMode(0755)|os.ModeDir, meta1.Mode,
				"Incorrect directory permissions for %s", forwardPath)

			// 列出父目录内容并验证
			parent := path.Dir(forwardPath)
			dirContents, err := store.List(ctx, parent)
			if err != nil {
				t.Fatalf("Failed to list parent directory %s: %v", parent, err)
			}

			// 验证目录在父目录列表中
			found := false
			expectedName := path.Base(forwardPath)
			for _, entry := range dirContents {
				if entry.Name == expectedName {
					found = true
					break
				}
			}
			assert.True(t, found, "Directory %s not found in parent directory listing", expectedName)
		})
	}
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{".", "/"},
		{"/", "/"},
		{"a", "/a"},
		{"a/", "/a"},
		{"/a", "/a"},
		{"/a/", "/a"},
		{"a/b", "/a/b"},
		{"a\\b", "/a/b"},
		{"\\a\\b", "/a/b"},
		{"a\\b/c", "/a/b/c"},
		{"../a", "/a"},
		{"a/../b", "/b"},
		{"a/./b", "/a/b"},
		{"./a/b", "/a/b"},
		{"a//b", "/a/b"},
		{"a\\/b", "/a/b"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Clean(%q)", tt.input), func(t *testing.T) {
			result := normalizePath(tt.input)
			assert.Equal(t, tt.expected, result, "Path cleaning failed")
		})
	}
}
