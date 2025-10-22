package global

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/gookit/goutil/dump"
	"github.com/haierkeys/singbox-subscribe-convert/pkg/fileurl"
	"gopkg.in/yaml.v3"
)

// Config 全局配置结构
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Auth    AuthConfig    `yaml:"auth"`
	Remote  RemoteConfig  `yaml:"remote"`
	Cache   CacheConfig   `yaml:"cache"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
	IdleTimeout  int `yaml:"idle_timeout"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Password string `yaml:"password"`
}

// RemoteConfig 远程文件配置
type RemoteConfig struct {
	NodeFileURL     string `yaml:"node_file_url"`
	TemplateURL     string `yaml:"template_url"`
	TemplateNoNode  string `yaml:"template_no_node"`
	RequestTimeout  int    `yaml:"request_timeout"`
	RefreshInterval int    `yaml:"refresh_interval"` // 分钟
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Directory    string `yaml:"directory"`
	NodeFile     string `yaml:"node_file"`
	TemplateFile string `yaml:"template_file"`
}

type LoggingConfig struct {
	// Level, See also zapcore.ParseLevel.
	Level string `yaml:"level"`

	// File that logger will be writen into.
	// Default is stderr.
	File string `yaml:"file"`

	// Production enables json output.
	Production bool `yaml:"production"`
	MaxSize    int  `yaml:"max_size"`
	MaxBackups int  `yaml:"max_backups"`
	MaxAge     int  `yaml:"max_age"`
}

var (
	// Cfg 全局配置实例
	Cfg *Config
	// ConfigFile 配置文件路径
	ConfigFile string
)

// Load 加载配置文件
func Load(configPath string) (string, error) {

	realpath, err := fileurl.GetAbsPath(configPath, "")
	if err != nil {
		return realpath, err
	}

	// 读取配置文件
	data, err := os.ReadFile(realpath)
	if err != nil {
		return "", fmt.Errorf("read config file error: %w", err)
	}

	// 解析 YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse config error: %w", err)
	}

	// 环境变量覆盖
	cfg.overrideWithEnv()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return "", fmt.Errorf("validate config error: %w", err)
	}

	Cfg = &cfg
	ConfigFile = configPath
	return realpath, nil
}

// overrideWithEnv 使用环境变量覆盖配置
func (c *Config) overrideWithEnv() {
	if val := os.Getenv("SERVER_PORT"); val != "" {
		fmt.Sscanf(val, "%d", &c.Server.Port)
	}
	if val := os.Getenv("PASSWORD"); val != "" {
		c.Auth.Password = val
	}
	if val := os.Getenv("NODE_FILE_URL"); val != "" {
		c.Remote.NodeFileURL = val
	}
	if val := os.Getenv("TEMPLATE_URL"); val != "" {
		c.Remote.TemplateURL = val
	}
	if val := os.Getenv("CACHE_DIR"); val != "" {
		c.Cache.Directory = val
	}
	if val := os.Getenv("REFRESH_INTERVAL"); val != "" {
		fmt.Sscanf(val, "%d", &c.Remote.RefreshInterval)
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Auth.Password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	if c.Remote.NodeFileURL == "" {
		return fmt.Errorf("node_file_url cannot be empty")
	}
	if c.Remote.TemplateURL == "" {
		return fmt.Errorf("template_url cannot be empty")
	}
	if c.Cache.Directory == "" {
		return fmt.Errorf("cache directory cannot be empty")
	}
	if c.Remote.RefreshInterval <= 0 {
		return fmt.Errorf("refresh_interval must be greater than 0")
	}
	return nil
}

// GetNodeFilePath 获取节点文件缓存路径
func (c *Config) GetNodeFilePath() string {
	return filepath.Join(c.Cache.Directory, c.Cache.NodeFile)
}

// GetTemplateFilePath 获取模板文件缓存路径
func (c *Config) GetTemplateFilePath() string {
	return filepath.Join(c.Cache.Directory, c.Cache.TemplateFile)
}

// GetLogFilePath 获取日志文件路径
func (c *Config) GetLogFilePath() string {
	return c.Logging.File
}

// GetRefreshInterval 获取刷新间隔
func (c *Config) GetRefreshInterval() time.Duration {
	return time.Duration(c.Remote.RefreshInterval) * time.Minute
}

// GetRequestTimeout 获取请求超时
func (c *Config) GetRequestTimeout() time.Duration {
	return time.Duration(c.Remote.RequestTimeout) * time.Second
}

// GetServerReadTimeout 获取服务器读取超时
func (c *Config) GetServerReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeout) * time.Second
}

// GetServerWriteTimeout 获取服务器写入超时
func (c *Config) GetServerWriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeout) * time.Second
}

// GetServerIdleTimeout 获取服务器空闲超时
func (c *Config) GetServerIdleTimeout() time.Duration {
	return time.Duration(c.Server.IdleTimeout) * time.Second
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config error: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file error: %w", err)
	}

	return nil
}
