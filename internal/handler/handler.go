package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/haierkeys/singbox-subscribe-convert/global"
	"github.com/haierkeys/singbox-subscribe-convert/internal/fetcher"
	"github.com/haierkeys/singbox-subscribe-convert/pkg/util"

	"github.com/flosch/pongo2/v6"

	"go.uber.org/zap"
)

var (
	cfg       *global.Config
	logger    *zap.Logger
	nodesName []string
	nodesData []map[string]interface{}
	nodes     []string
	template  *pongo2.Template
	dataMutex sync.RWMutex
)

// NodeFile 节点文件结构
type NodeFile struct {
	Outbounds []map[string]interface{} `json:"outbounds"`
}

// Init 初始化 handler
func Init(c *global.Config, l *zap.Logger) error {
	cfg = c
	logger = l

	// 注册自定义过滤器
	pongo2.RegisterFilter("NotesName", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

		paramStr := ""
		if in != nil {
			paramStr = in.String()
		}
		result := nodeNameFilter(paramStr)
		return pongo2.AsSafeValue(result), nil
	})

	if err := ReloadData(); err != nil {
		logger.Warn("Failed to load initial data",
			zap.Error(err),
		)
	}

	if err := ReloadTemplate(); err != nil {
		logger.Warn("Failed to load initial template",
			zap.Error(err),
		)
	}

	return nil
}

// ReloadData 重新加载节点数据
func ReloadData() error {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	nodeFilePath := cfg.GetNodeFilePath()

	if _, err := os.Stat(nodeFilePath); os.IsNotExist(err) {
		return fmt.Errorf("node file not found: %s", nodeFilePath)
	}

	data, err := os.ReadFile(nodeFilePath)
	if err != nil {
		return fmt.Errorf("read node file error: %w", err)
	}

	var nodeFile NodeFile
	if err := json.Unmarshal(data, &nodeFile); err != nil {
		return fmt.Errorf("parse node file error: %w", err)
	}

	if nodeFile.Outbounds == nil || len(nodeFile.Outbounds) == 0 {
		return fmt.Errorf("no outbounds found in node file")
	}

	nodesName = []string{}
	nodesData = make([]map[string]interface{}, 0)
	nodes = []string{}

	// 提取所有节点的 tag
	for _, node := range nodeFile.Outbounds {
		if tag, ok := node["tag"].(string); ok {
			if !util.InSlice(nodesName, tag) {
				nodesName = append(nodesName, tag)
				nodesData = append(nodesData, node)

				nodeStr, _ := json.Marshal(node)
				nodes = append(nodes, string(nodeStr))
			}
		}
	}

	logger.Info("✓ Loaded node data",
		zap.String("file_path", nodeFilePath),
		zap.Int("outbounds", len(nodesName)),
	)
	return nil
}

// ReloadTemplate 重新加载模板
func ReloadTemplate() error {
	dataMutex.Lock()
	defer dataMutex.Unlock()
	templateFilePath := cfg.GetTemplateFilePath()
	if _, err := os.Stat(templateFilePath); os.IsNotExist(err) {
		return fmt.Errorf("template file not found: %s", templateFilePath)
	}
	tpl, err := pongo2.FromFile(templateFilePath)
	if err != nil {
		return fmt.Errorf("load template error: %w", err)
	}
	template = tpl
	logger.Info("✓ Loaded template from cache",
		zap.String("file_path", templateFilePath),
	)
	return nil
}

// HandleRequest 处理主请求
func HandleRequest(w http.ResponseWriter, r *http.Request) {
	logger.Info("Request received",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("path", r.URL.Path),
	)
	queryParams := r.URL.Query()
	setType := queryParams.Get("type")
	password := queryParams.Get("password")
	if password != cfg.Auth.Password {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Password Error"))
		logger.Warn("Unauthorized request",
			zap.String("remote_addr", r.RemoteAddr),
		)
		return
	}
	dataMutex.RLock()
	currentTemplate := template
	dataMutex.RUnlock()
	if currentTemplate == nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Template not loaded"))
		return
	}
	// 构建模板上下文
	context := pongo2.Context{
		"Nodes":     pongo2.AsSafeValue(strings.Join(nodes, ",\r\n")),
		"setType":   setType,
		"nodeCount": len(nodes),
	}
	// tpl, _ := pongo2.FromString("{{ Nodes }}111111")
	// output, err := tpl.Execute(context)
	output, err := currentTemplate.Execute(context)
	if err != nil {
		logger.Error("Error rendering template",
			zap.Error(err),
		)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Server Error: %v", err)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Profile-Update-Interval", "6")
	w.Header().Set("Subscription-Userinfo", fmt.Sprintf("upload=0; download=0; total=%d", len(nodes)))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
	logger.Info("Successfully served config",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("type", setType),
		zap.Int("node_count", len(nodes)),
	)
}

// HandleHealth 健康检查
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	dataMutex.RLock()
	hasData := len(nodesData) > 0
	hasTemplate := template != nil
	nodeCount := len(nodesData)
	dataMutex.RUnlock()
	status := "ok"
	code := http.StatusOK
	if !hasData || !hasTemplate {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"status":"%s","has_data":%t,"has_template":%t,"node_count":%d}`,
		status, hasData, hasTemplate, nodeCount)
}

// HandleRefresh 手动刷新
func HandleRefresh(w http.ResponseWriter, r *http.Request) {
	password := r.URL.Query().Get("password")
	if password != cfg.Auth.Password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Password Error"))
		return
	}
	logger.Info("Manual refresh triggered",
		zap.String("remote_addr", r.RemoteAddr),
	)
	errChan := make(chan error, 2)
	go func() {
		if err := fetcher.FetchNodeFile(); err != nil {
			errChan <- fmt.Errorf("node file: %w", err)
		} else {
			errChan <- ReloadData()
		}
	}()
	go func() {
		if err := fetcher.FetchTemplateFile(); err != nil {
			errChan <- fmt.Errorf("template file: %w", err)
		} else {
			errChan <- ReloadTemplate()
		}
	}()
	var errors []string
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err.Error())
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if len(errors) > 0 {
		w.WriteHeader(http.StatusInternalServerError)
		errJSON, _ := json.Marshal(errors)
		fmt.Fprintf(w, `{"status":"error","errors":%s}`, string(errJSON))
		logger.Error("Manual refresh failed",
			zap.Strings("errors", errors),
		)
	} else {
		dataMutex.RLock()
		nodeCount := len(nodesData)
		dataMutex.RUnlock()
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"success","message":"Files refreshed successfully","node_count":%d}`, nodeCount)
		logger.Info("Manual refresh completed successfully")
	}
}

// nodeNameFilter 过滤节点名称
func nodeNameFilter(param string) string {
	dataMutex.RLock()
	defer dataMutex.RUnlock()
	filteredList := []string{}
	if param == "" {
		// 如果没有参数,返回所有节点名
		filteredList = nodesName
	} else {
		// 按照 | 分隔的参数进行过滤
		nameParams := strings.Split(param, "|")
		for _, nodeName := range nodesName {
			for _, name := range nameParams {
				name = strings.TrimSpace(name)
				if name != "" && strings.Contains(nodeName, name) {
					filteredList = append(filteredList, nodeName)
					break
				}
			}
		}
	}
	if len(filteredList) == 0 {
		filteredList = append(filteredList, cfg.Remote.TemplateNoNode)
	}
	jsonBytes, _ := json.Marshal(filteredList)
	s := string(jsonBytes)
	// 去掉外层的 []
	if len(s) > 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1]
	}
	return s
}
