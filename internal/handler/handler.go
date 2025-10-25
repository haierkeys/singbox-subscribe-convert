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
	templates map[string]*pongo2.Template
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

	// 初始化模板映射
	templates = make(map[string]*pongo2.Template)

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

	if err := ReloadAllTemplates(); err != nil {
		logger.Warn("Failed to load initial templates",
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

// ReloadTemplateByName 根据名称重新加载模板
func ReloadTemplateByName(templateName string) error {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	templateFilePath := cfg.GetTemplateFilePathByName(templateName)
	if _, err := os.Stat(templateFilePath); os.IsNotExist(err) {
		return fmt.Errorf("template file not found: %s", templateFilePath)
	}

	tpl, err := pongo2.FromFile(templateFilePath)
	if err != nil {
		return fmt.Errorf("load template error: %w", err)
	}

	templates[templateName] = tpl
	logger.Info("✓ Loaded template from cache",
		zap.String("template", templateName),
		zap.String("file_path", templateFilePath),
	)
	return nil
}

// ReloadAllTemplates 重新加载所有启用的模板
func ReloadAllTemplates() error {
	enabledTemplates := cfg.GetEnabledTemplates()
	var errors []string

	for name := range enabledTemplates {
		if err := ReloadTemplateByName(name); err != nil {
			logger.Error("Failed to load template",
				zap.String("template", name),
				zap.Error(err),
			)
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some templates: %s", strings.Join(errors, "; "))
	}

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
	templateName := queryParams.Get("template") // 新增：支持指定模板

	if password != cfg.Auth.Password {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Password Error"))
		logger.Warn("Unauthorized request",
			zap.String("remote_addr", r.RemoteAddr),
		)
		return
	}

	// 获取要使用的模板
	if templateName == "" {
		templateName = cfg.DefaultTemplate
	}

	dataMutex.RLock()
	var currentTemplate *pongo2.Template
	var actualTemplateName string
	var noNodeName string

	// 检查模板是否启用
	if tplConfig, exists := cfg.GetTemplate(templateName); exists && tplConfig.Enabled {
		currentTemplate = templates[templateName]
		actualTemplateName = tplConfig.Name
		noNodeName = tplConfig.NoNode
	} else {
		dataMutex.RUnlock()
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Template '%s' not found or not enabled", templateName)))
		logger.Warn("Template not found or not enabled",
			zap.String("template", templateName),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return
	}
	dataMutex.RUnlock()

	if currentTemplate == nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Template '%s' not loaded", templateName)))
		return
	}

	// 构建模板上下文
	context := pongo2.Context{
		"Nodes":     pongo2.AsSafeValue(strings.Join(nodes, ",\r\n")),
		"setType":   setType,
		"nodeCount": len(nodes),
		"noNode":    noNodeName,
	}

	output, err := currentTemplate.Execute(context)
	if err != nil {
		logger.Error("Error rendering template",
			zap.Error(err),
			zap.String("template", templateName),
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
		zap.String("template", templateName),
		zap.String("template_name", actualTemplateName),
		zap.String("type", setType),
		zap.Int("node_count", len(nodes)),
	)
}

// HandleHealth 健康检查
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	dataMutex.RLock()
	hasData := len(nodesData) > 0
	hasTemplate := len(templates) > 0
	templateCount := len(templates)
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
	fmt.Fprintf(w, `{"status":"%s","has_data":%t,"has_template":%t,"node_count":%d,"template_count":%d}`,
		status, hasData, hasTemplate, nodeCount, templateCount)
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

	var errors []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 刷新节点文件
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fetcher.FetchNodeFile(); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Sprintf("node file: %v", err))
			mu.Unlock()
		} else {
			if err := ReloadData(); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("reload node data: %v", err))
				mu.Unlock()
			}
		}
	}()

	// 刷新所有启用的模板
	enabledTemplates := cfg.GetEnabledTemplates()
	for name, tpl := range enabledTemplates {
		wg.Add(1)
		go func(templateName string, templateURL string) {
			defer wg.Done()
			if err := fetcher.FetchTemplateFileByName(templateName, templateURL); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("template %s: %v", templateName, err))
				mu.Unlock()
			} else {
				if err := ReloadTemplateByName(templateName); err != nil {
					mu.Lock()
					errors = append(errors, fmt.Sprintf("reload template %s: %v", templateName, err))
					mu.Unlock()
				}
			}
		}(name, tpl.URL)
	}

	wg.Wait()

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
		templateCount := len(templates)
		dataMutex.RUnlock()

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"success","message":"Files refreshed successfully","node_count":%d,"template_count":%d}`, nodeCount, templateCount)
		logger.Info("Manual refresh completed successfully",
			zap.Int("node_count", nodeCount),
			zap.Int("template_count", templateCount),
		)
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
		// 使用配置的无节点标识
		noNodeName := cfg.GetDefaultTemplateNoNode()
		filteredList = append(filteredList, noNodeName)
	}

	jsonBytes, _ := json.Marshal(filteredList)
	s := string(jsonBytes)
	// 去掉外层的 []
	if len(s) > 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1]
	}
	return s
}
