package meta

import (
	"context"
	"os"
	"time"
)

// FileType 定义文件类型
type FileType int

const (
	TypeRegular   FileType = iota // 普通文件
	TypeDirectory                 // 目录
	TypeSymlink                   // 符号链接
)

// Metadata 文件元数据
type Metadata struct {
	Inode      uint64      `json:"inode"`       // Inode号
	Name       string      `json:"name"`        // 文件名
	Type       FileType    `json:"type"`        // 文件类型
	Size       int64       `json:"size"`        // 文件大小
	Mode       os.FileMode `json:"mode"`        // 文件权限
	Blocks     []Block     `json:"blocks"`      // 数据块列表
	Links      int         `json:"links"`       // 硬链接数
	Owner      string      `json:"owner"`       // 所有者
	Group      string      `json:"group"`       // 组
	CreateTime time.Time   `json:"create_time"` // 创建时间
	ModifyTime time.Time   `json:"modify_time"` // 修改时间
	AccessTime time.Time   `json:"access_time"` // 访问时间
	Version    uint64      `json:"version"`     // 版本号
}

// Block 数据块信息
type Block struct {
	ID        string   `json:"id"`        // 块ID
	Size      int64    `json:"size"`      // 块大小
	Offset    int64    `json:"offset"`    // 文件内偏移
	Checksum  string   `json:"checksum"`  // 校验和
	Locations []string `json:"locations"` // 数据服务器位置
}

// MetaStore 元数据存储接口
type MetaStore interface {
	// 文件操作
	Create(ctx context.Context, path string, mode os.FileMode) (*Metadata, error)
	Get(ctx context.Context, path string) (*Metadata, error)
	Update(ctx context.Context, path string, meta *Metadata) error
	Delete(ctx context.Context, path string) error

	// 目录操作
	List(ctx context.Context, path string) ([]*Metadata, error)
	Mkdir(ctx context.Context, path string, mode os.FileMode) error

	// 事务操作
	Begin() (Transaction, error)

	// 快照操作
	CreateSnapshot(ctx context.Context, path string) (string, error)
	RestoreSnapshot(ctx context.Context, snapshotID string) error
}

// Transaction 事务接口
type Transaction interface {
	Commit() error
	Rollback() error
}
