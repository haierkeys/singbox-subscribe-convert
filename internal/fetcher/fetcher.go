package fetcher

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/haierkeys/singbox-subscribe-convert/global"
)

var (
	cfg        *global.Config
	logger     *zap.Logger
	httpClient *http.Client
)

// Init 初始化 fetcher
func Init(c *global.Config, l *zap.Logger) {
	cfg = c
	logger = l

	httpClient = &http.Client{
		Timeout: cfg.GetRequestTimeout(),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// fetchFile 从 URL 获取文件并保存
func fetchFile(url, cachePath string) error {
	// 添加随机数参数以绕过 CDN 缓存
	urlWithParam := addCacheBusterParam(url)
	logger.Info("Fetching file from %s", zap.String("url", urlWithParam))

	req, err := http.NewRequest("GET", urlWithParam, nil)
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("User-Agent", "Singbox-Subscribe-Convert/1.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response error: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("received empty file")
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("create cache dir error: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("write cache file error: %w", err)
	}

	logger.Info("Successfully fetched and cached: %s (%d bytes)", zap.String("cachePath", cachePath), zap.Int("len", len(data)))
	return nil
}

// FetchNodeFile 获取节点文件
func FetchNodeFile() error {
	return fetchFile(cfg.Subscription.URL, cfg.GetNodeFilePath())
}

// FetchTemplateFileByName 根据模板名称获取模板文件
func FetchTemplateFileByName(templateName string, templateURL string) error {
	cachePath := cfg.GetTemplateFilePathByName(templateName)
	return fetchFile(templateURL, cachePath)
}

// FetchAllTemplates 获取所有启用的模板文件
func FetchAllTemplates() map[string]error {
	errors := make(map[string]error)

	// 获取所有启用的模板
	enabledTemplates := cfg.GetEnabledTemplates()
	for name, tpl := range enabledTemplates {
		if err := FetchTemplateFileByName(name, tpl.URL); err != nil {
			logger.Error("Failed to fetch template",
				zap.String("template", name),
				zap.String("url", tpl.URL),
				zap.Error(err),
			)
			errors[name] = err
		} else {
			logger.Info("Successfully fetched template",
				zap.String("template", name),
				zap.String("name", tpl.Name),
			)
		}
	}
	return errors
}

// CheckCacheExists 检查缓存是否存在
func CheckCacheExists() bool {
	nodeExists := fileExists(cfg.GetNodeFilePath())
	defaultTemplatePath := cfg.GetTemplateFilePathByName(cfg.DefaultTemplate)
	return nodeExists && fileExists(defaultTemplatePath)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

// addCacheBusterParam 给 URL 添加随机数参数以绕过 CDN 缓存
func addCacheBusterParam(url string) string {
	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}
	// 使用时间戳纳秒作为随机参数
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s%s_t=%d", url, separator, timestamp)
}
