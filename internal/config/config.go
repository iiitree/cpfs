package config

import (
	"github.com/spf13/viper"
)

// ServerConfig 定义服务器配置结构
type ServerConfig struct {
	// 基础配置
	ServerID      string `mapstructure:"server_id"`
	ServerType    string `mapstructure:"server_type"` // meta/data
	ListenAddress string `mapstructure:"listen_address"`
	DataDir       string `mapstructure:"data_dir"`

	// 元数据服务器配置
	MetaServers []string `mapstructure:"meta_servers"`

	// 数据服务器配置
	DataServers []string `mapstructure:"data_servers"`

	// RAID配置
	RaidLevel  int   `mapstructure:"raid_level"`
	StripeSize int64 `mapstructure:"stripe_size"`

	// 高可用配置
	HeartbeatInterval int `mapstructure:"heartbeat_interval"`
	FailureTimeout    int `mapstructure:"failure_timeout"`

	// 缓存配置
	CacheSize int64 `mapstructure:"cache_size"`
	CacheTTL  int   `mapstructure:"cache_ttl"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*ServerConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &ServerConfig{}
	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
