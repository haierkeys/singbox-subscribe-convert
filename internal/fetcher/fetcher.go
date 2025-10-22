package fetcher

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	logger.Info("Fetching file from %s", zap.String("url", url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("User-Agent", "Singbox-Subscribe-Convert/1.0")
	req.Header.Set("Accept", "*/*")

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
	return fetchFile(cfg.Remote.NodeFileURL, cfg.GetNodeFilePath())
}

// FetchTemplateFile 获取模板文件
func FetchTemplateFile() error {
	return fetchFile(cfg.Remote.TemplateURL, cfg.GetTemplateFilePath())
}

// CheckCacheExists 检查缓存是否存在
func CheckCacheExists() bool {
	nodeExists := fileExists(cfg.GetNodeFilePath())
	templateExists := fileExists(cfg.GetTemplateFilePath())
	return nodeExists && templateExists
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}
