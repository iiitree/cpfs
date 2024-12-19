package meta

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDir 创建临时测试目录
func setupTestDir(tb testing.TB) string {
	tb.Helper()
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	require.NoError(tb, err)
	return tempDir
}

// TestFileStorage 测试文件存储的基本功能
func TestFileStorage(t *testing.T) {
	tempDir := setupTestDir(t)
	defer os.RemoveAll(tempDir)

	config := &StorageConfig{
		RootDir:           tempDir,
		SyncInterval:      time.Millisecond * 100,
		FileMode:          0644,
		EnableCompression: false,
	}

	storage, err := NewFileStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// 基本操作测试
	t.Run("Basic Operations", func(t *testing.T) {
		key := "/test/basic.txt"
		data := []byte("Hello, World!")

		err := storage.Save(ctx, key, data)
		assert.NoError(t, err)

		time.Sleep(config.SyncInterval * 2)

		path, err := storage.keyToPath(strings.TrimPrefix(key, "/"))
		require.NoError(t, err)
		_, err = os.Stat(path)
		assert.NoError(t, err)

		loaded, err := storage.Load(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, data, loaded)

		err = storage.Delete(ctx, key)
		assert.NoError(t, err)

		_, err = storage.Load(ctx, key)
		assert.Error(t, err)
	})

	// 目录结构测试
	t.Run("Directory Structure", func(t *testing.T) {
		keys := []string{
			"/dir1/file1.txt",
			"/dir1/subdir/file2.txt",
			"/dir2/file3.txt",
		}
		data := []byte("test data")

		for _, key := range keys {
			err := storage.Save(ctx, key, data)
			assert.NoError(t, err)
		}

		time.Sleep(config.SyncInterval * 2)

		list, err := storage.List(ctx, "/dir1")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(list))

		for _, key := range keys {
			path, err := storage.keyToPath(strings.TrimPrefix(key, "/"))
			require.NoError(t, err)
			_, err = os.Stat(path)
			assert.NoError(t, err)
		}
	})

	// 文件更新测试
	t.Run("File Updates", func(t *testing.T) {
		key := "/test/update.txt"
		data1 := []byte("version 1")
		data2 := []byte("version 2")

		err := storage.Save(ctx, key, data1)
		assert.NoError(t, err)

		time.Sleep(config.SyncInterval)

		err = storage.Save(ctx, key, data2)
		assert.NoError(t, err)

		time.Sleep(config.SyncInterval)

		loaded, err := storage.Load(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, data2, loaded)
	})

	// 并发操作测试
	t.Run("Concurrent Operations", func(t *testing.T) {
		const goroutines = 10
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("/concurrent/file%d-%d.txt", id, j)
					data := []byte(fmt.Sprintf("data-%d-%d", id, j))

					err := storage.Save(ctx, key, data)
					assert.NoError(t, err)

					loaded, err := storage.Load(ctx, key)
					if err == nil {
						assert.Equal(t, data, loaded)
					}

					_ = storage.Delete(ctx, key)
				}
			}(i)
		}

		wg.Wait()
	})

	// 错误处理测试
	t.Run("Error Handling", func(t *testing.T) {
		// 测试不存在的文件
		_, err := storage.Load(ctx, "/nonexistent.txt")
		assert.Error(t, err)

		// 测试空键
		err = storage.Save(ctx, "", []byte("test"))
		assert.Error(t, err)

		// 测试根路径
		err = storage.Save(ctx, "/", []byte("test"))
		assert.Error(t, err)

		// Windows特定测试
		if runtime.GOOS == "windows" {
			err = storage.Save(ctx, "/test/invalid\x00char.txt", []byte("test"))
			assert.Error(t, err)
		}
	})
}

// 基准测试部分
// 最简单的基准测试
func BenchmarkSimple(b *testing.B) {
	b.Log("Starting simple benchmark")
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("test-%d", i)
	}
}

// 文件保存基准测试
func BenchmarkStorageSave(b *testing.B) {
	tempDir := setupTestDir(b)
	defer os.RemoveAll(tempDir)

	fs, err := NewFileStorage(&StorageConfig{
		RootDir:      tempDir,
		SyncInterval: time.Second,
		FileMode:     0644,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer fs.Close()

	ctx := context.Background()
	data := []byte("benchmark test data")

	b.Log("Starting storage save benchmark")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := fs.Save(ctx, fmt.Sprintf("test/file-%d.txt", i), data); err != nil {
			b.Fatal(err)
		}
	}
}

// 文件加载基准测试
func BenchmarkStorageLoad(b *testing.B) {
	tempDir := setupTestDir(b)
	defer os.RemoveAll(tempDir)

	fs, err := NewFileStorage(&StorageConfig{
		RootDir:      tempDir,
		SyncInterval: time.Second,
		FileMode:     0644,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer fs.Close()

	ctx := context.Background()
	data := []byte("benchmark test data")
	key := "test/benchmark-load.txt"

	err = fs.Save(ctx, key, data)
	if err != nil {
		b.Fatal(err)
	}

	b.Log("Starting storage load benchmark")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := fs.Load(ctx, key); err != nil {
			b.Fatal(err)
		}
	}
}

// 文件列表基准测试
func BenchmarkStorageList(b *testing.B) {
	tempDir := setupTestDir(b)
	defer os.RemoveAll(tempDir)

	fs, err := NewFileStorage(&StorageConfig{
		RootDir:      tempDir,
		SyncInterval: time.Second,
		FileMode:     0644,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer fs.Close()

	ctx := context.Background()
	data := []byte("benchmark test data")

	// 准备测试数据
	for i := 0; i < 10; i++ {
		err := fs.Save(ctx, fmt.Sprintf("test/list/file-%d.txt", i), data)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.Log("Starting storage list benchmark")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := fs.List(ctx, "test/list/"); err != nil {
			b.Fatal(err)
		}
	}
}

// 并发操作基准测试
func BenchmarkStorageConcurrent(b *testing.B) {
	tempDir := setupTestDir(b)
	defer os.RemoveAll(tempDir)

	fs, err := NewFileStorage(&StorageConfig{
		RootDir:      tempDir,
		SyncInterval: time.Second,
		FileMode:     0644,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer fs.Close()

	ctx := context.Background()
	data := []byte("benchmark test data")

	b.Log("Starting concurrent storage operations benchmark")
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("test/concurrent-%d.txt", counter)
			if err := fs.Save(ctx, key, data); err != nil {
				b.Fatal(err)
			}
			counter++
		}
	})
}
