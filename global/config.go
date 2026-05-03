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
	Server          ServerConfig              `yaml:"server"`
	Auth            AuthConfig                `yaml:"auth"`
	Subscription    SubscriptionConfig        `yaml:"subscription"`
	Templates       map[string]TemplateConfig `yaml:"templates"`
	DefaultTemplate string                    `yaml:"default_template"`
	Cache           CacheConfig               `yaml:"cache"`
	Cloudflare      CloudflareConfig          `yaml:"cloudflare"`
	Logging         LoggingConfig             `yaml:"logging"`
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

// SubscriptionConfig 订阅配置
type SubscriptionConfig struct {
	URL             string `yaml:"url"`
	Timeout         int    `yaml:"timeout"`          // 秒
	RefreshInterval int    `yaml:"refresh_interval"` // 分钟
}

// CloudflareConfig Cloudflare 配置
type CloudflareConfig struct {
	PurgeURL string `yaml:"purge_url"` // Cloudflare 缓存清理 API 地址
	Enabled  bool   `yaml:"enabled"`   // 是否启用 Cloudflare 缓存清理
	APIToken string `yaml:"api_token"` // Cloudflare API Token (推荐使用)
	APIKey   string `yaml:"api_key"`   // Cloudflare API Key (可选，与 api_email 一起使用)
	APIEmail string `yaml:"api_email"` // Cloudflare API Email (与 api_key 一起使用)
}

// TemplateConfig 模板配置
type TemplateConfig struct {
	URL            string `yaml:"url"`
	Name           string `yaml:"name"`
	NoNode         string `yaml:"no_node"`
	Enabled        bool   `yaml:"enabled"`
	UpdateInterval int    `yaml:"update_interval"` // 秒
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
	if val := os.Getenv("SUBSCRIPTION_URL"); val != "" {
		c.Subscription.URL = val
	}
	if val := os.Getenv("DEFAULT_TEMPLATE"); val != "" {
		c.DefaultTemplate = val
	}
	if val := os.Getenv("CACHE_DIR"); val != "" {
		c.Cache.Directory = val
	}
	if val := os.Getenv("REFRESH_INTERVAL"); val != "" {
		fmt.Sscanf(val, "%d", &c.Subscription.RefreshInterval)
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
	if c.Cache.Directory == "" {
		return fmt.Errorf("cache directory cannot be empty")
	}
	if c.Subscription.URL == "" {
		return fmt.Errorf("subscription url cannot be empty")
	}
	if c.Subscription.RefreshInterval <= 0 {
		return fmt.Errorf("subscription refresh_interval must be greater than 0")
	}
	if len(c.Templates) == 0 {
		return fmt.Errorf("at least one template must be configured")
	}
	if c.DefaultTemplate == "" {
		return fmt.Errorf("default_template cannot be empty")
	}
	if _, exists := c.Templates[c.DefaultTemplate]; !exists {
		return fmt.Errorf("default_template '%s' not found in templates", c.DefaultTemplate)
	}

	// 验证至少有一个启用的模板
	hasEnabled := false
	for _, tpl := range c.Templates {
		if tpl.Enabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		return fmt.Errorf("at least one template must be enabled")
	}

	return nil
}

// GetNodeFilePath 获取节点文件缓存路径
func (c *Config) GetNodeFilePath() string {
	return filepath.Join(c.Cache.Directory, c.Cache.NodeFile)
}

// GetTemplateFilePathByName 根据模板名称获取模板文件缓存路径
func (c *Config) GetTemplateFilePathByName(templateName string) string {
	return filepath.Join(c.Cache.Directory, fmt.Sprintf("template_%s.json", templateName))
}

// GetEnabledTemplates 获取所有启用的模板
func (c *Config) GetEnabledTemplates() map[string]TemplateConfig {
	enabled := make(map[string]TemplateConfig)
	for name, tpl := range c.Templates {
		if tpl.Enabled {
			enabled[name] = tpl
		}
	}
	return enabled
}

// GetTemplate 根据名称获取模板配置
func (c *Config) GetTemplate(name string) (TemplateConfig, bool) {
	tpl, exists := c.Templates[name]
	return tpl, exists
}

// GetDefaultTemplateNoNode 获取默认模板的无节点标识
func (c *Config) GetDefaultTemplateNoNode() string {
	if tpl, exists := c.Templates[c.DefaultTemplate]; exists {
		return tpl.NoNode
	}
	return "🎯 全球直连"
}
// GetTemplateUpdateInterval 获取模板更新间隔
func (c *Config) GetTemplateUpdateInterval(templateName string) time.Duration {
	if tpl, exists := c.Templates[templateName]; exists && tpl.UpdateInterval > 0 {
		return time.Duration(tpl.UpdateInterval) * time.Second
	}
	// 默认 1 小时
	return 1 * time.Hour
}

// GetLogFilePath 获取日志文件路径
func (c *Config) GetLogFilePath() string {
	return c.Logging.File
}

// GetRefreshInterval 获取刷新间隔
func (c *Config) GetRefreshInterval() time.Duration {
	return time.Duration(c.Subscription.RefreshInterval) * time.Minute
}

// GetRequestTimeout 获取请求超时
func (c *Config) GetRequestTimeout() time.Duration {
	if c.Subscription.Timeout > 0 {
		return time.Duration(c.Subscription.Timeout) * time.Second
	}
	return 30 * time.Second
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
