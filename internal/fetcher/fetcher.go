package fetcher

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
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
	fmt.Println("\n==================================================")
	fmt.Printf("🚀 DOWNLOAD URL: %s\n", urlWithParam)
	fmt.Println("==================================================\n")
	logger.Info("🚀 [DOWNLOAD] Starting fetch from URL: " + urlWithParam)
	// 同时向 Stderr 打印一份，防止 Stdout 被缓冲
	fmt.Fprintf(os.Stderr, "[FETCHING] %s\n", urlWithParam)

	req, err := http.NewRequest("GET", urlWithParam, nil)
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	// Use browser User-Agent to prevent subscription conversion server from returning degraded nodes.
	// 使用浏览器 User-Agent，防止订阅转换站返回降级节点。
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
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
	nodeExists := IsFileExists(cfg.GetNodeFilePath())
	defaultTemplatePath := cfg.GetTemplateFilePathByName(cfg.DefaultTemplate)
	return nodeExists && IsFileExists(defaultTemplatePath)
}

func IsFileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

// GetFileModTime 获取文件修改时间
func GetFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// addCacheBusterParam 给 URL 添加随机数参数以绕过 CDN 缓存
func addCacheBusterParam(url string) string {
	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}

	// 生成 4 字节的随机十六进制字符串 (8个字符)
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	randomStr := hex.EncodeToString(b)

	// 使用时间戳和随机字符串
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s%s_t=%d&_r=%s", url, separator, timestamp, randomStr)
}
