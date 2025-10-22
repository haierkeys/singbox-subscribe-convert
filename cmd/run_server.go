package cmd

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/haierkeys/singbox-subscribe-convert/global"
	"github.com/haierkeys/singbox-subscribe-convert/internal/fetcher"
	"github.com/haierkeys/singbox-subscribe-convert/internal/handler"
	"github.com/haierkeys/singbox-subscribe-convert/internal/watcher"
	"github.com/haierkeys/singbox-subscribe-convert/pkg/fileurl"
	"github.com/haierkeys/singbox-subscribe-convert/pkg/logger"
	"github.com/haierkeys/singbox-subscribe-convert/pkg/safe_close"

	"go.uber.org/zap"
)

// Server 服务器主结构体
type Server struct {
	logger     *zap.Logger           // 日志记录器
	httpServer *http.Server          // HTTP 服务器实例
	sc         *safe_close.SafeClose // 安全关闭管理器
	ctx        context.Context       // 上下文，用于控制后台任务
	cancel     context.CancelFunc    // 取消函数，用于停止后台任务
}

// NewServer 创建并初始化服务器实例
func NewServer(runEnv *runFlags) (*Server, error) {
	s := &Server{
		sc: safe_close.NewSafeClose(),
	}
	// 创建可取消的上下文，用于管理后台 goroutine 的生命周期
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// 初始化临时 logger（在配置加载前使用）
	tempLogger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create temp logger: %w", err)
	}

	// 确保临时 logger 在函数返回前刷新缓冲
	defer func() {
		_ = tempLogger.Sync()
	}()

	// 加载配置文件
	configRealpath, err := global.Load(runEnv.config)
	if err != nil {
		tempLogger.Error("Error loading config", zap.Error(err))
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 使用配置初始化正式的 logger
	if err := s.initLogger(); err != nil {
		tempLogger.Error("Failed to initialize logger", zap.Error(err))
		return nil, err
	}

	cfg := global.Cfg

	// 记录服务器启动信息
	s.logStartupInfo(configRealpath, cfg)

	// 创建缓存目录（如果不存在）
	if err := fileurl.CreatePath(cfg.Cache.Directory, 0755); err != nil {
		return nil, err
	}

	// 初始化文件获取器（fetcher）
	fetcher.Init(cfg, s.logger)

	// 初始化数据（首次获取远程文件或使用缓存）
	if err := s.initializeData(); err != nil {
		s.logger.Warn("Data initialization had issues", zap.Error(err))
		fmt.Println("⚠ Warning: Data initialization incomplete, server may not work properly")
	}

	// 初始化请求处理器（handler）
	if err := handler.Init(cfg, s.logger); err != nil {
		s.logger.Error("Failed to initialize handler", zap.Error(err))
		return nil, fmt.Errorf("handler init failed: %w", err)
	}

	// 启动后台服务（自动更新、文件监控）
	s.startBackgroundServices(cfg)

	// 启动 HTTP 服务器
	if err := s.startHTTPServer(cfg); err != nil {
		return nil, err
	}

	return s, nil
}

// initLogger 初始化日志系统
func (s *Server) initLogger() error {
	// 如果配置了日志文件且目录不存在，则创建目录
	if global.Cfg.Logging.File != "" && !fileurl.IsExist(global.Cfg.Logging.File) {
		if err := fileurl.CreatePath(global.Cfg.Logging.File, 0755); err != nil {
			return err
		}
	}

	// 根据配置创建 logger
	lg, err := logger.NewLogger(logger.Config{
		Level:      global.Cfg.Logging.Level,      // 日志级别
		File:       global.Cfg.Logging.File,       // 日志文件路径
		Production: global.Cfg.Logging.Production, // 是否生产模式
	})
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	// 设置全局 logger
	global.Logger = lg
	s.logger = lg
	return nil
}

// initializeData 初始化数据（首次获取或使用缓存）
func (s *Server) initializeData() error {
	// 检查缓存是否存在
	if !fetcher.CheckCacheExists() {
		s.logger.Info("Cache not found, performing initial fetch...")
		fmt.Println("Fetching remote files for the first time...")

		// 执行首次文件获取
		if err := s.performInitialFetch(); err != nil {
			s.logger.Warn("Initial fetch failed", zap.Error(err))
			return err
		}

		s.logger.Info("✓ Initial fetch completed successfully")
		fmt.Println("✓ Initial fetch completed successfully")
	} else {
		s.logger.Info("Using existing cache files")
	}
	return nil
}

// startBackgroundServices 启动后台服务
func (s *Server) startBackgroundServices(cfg *global.Config) {
	// 启动定期自动更新服务
	s.logger.Info("Starting auto-update service",
		zap.Duration("interval", cfg.GetRefreshInterval()),
	)
	go s.startAutoUpdate(cfg)

	// 启动配置文件监控服务（监控配置变化并自动重载）
	go watcher.Start(s.ctx, cfg, s.logger, handler.ReloadData, handler.ReloadTemplate)
}

// logStartupInfo 记录服务器启动信息
func (s *Server) logStartupInfo(configPath string, cfg *global.Config) {
	s.logger.Info("=== Singbox Subscribe Convert Server Starting ===")
	s.logger.Info("Server information",
		zap.String("name", global.Name),
		zap.String("version", global.Version),
		zap.String("git_tag", global.GitTag),
		zap.String("build_time", global.BuildTime),
	)
	s.logger.Info("Configuration loaded",
		zap.String("config_file", configPath),
		zap.Int("server_port", cfg.Server.Port),
		zap.String("node_file_url", cfg.Remote.NodeFileURL),
		zap.String("template_url", cfg.Remote.TemplateURL),
		zap.String("cache_directory", cfg.Cache.Directory),
		zap.Duration("auto_refresh_interval", cfg.GetRefreshInterval()),
	)
}

// startHTTPServer 启动 HTTP 服务器
func (s *Server) startHTTPServer(cfg *global.Config) error {
	// 验证端口配置
	if cfg.Server.Port <= 0 {
		return fmt.Errorf("cfg.Server.Port Error")
	}

	// 注册路由
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.HandleRequest)        // 主要订阅转换接口
	mux.HandleFunc("/health", handler.HandleHealth)   // 健康检查接口
	mux.HandleFunc("/refresh", handler.HandleRefresh) // 手动刷新接口

	// 创建 HTTP 服务器
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.GetServerReadTimeout(),  // 读取超时
		WriteTimeout: cfg.GetServerWriteTimeout(), // 写入超时
		IdleTimeout:  cfg.GetServerIdleTimeout(),  // 空闲超时
	}

	s.logger.Info("✓ Server is running",
		zap.Int("port", cfg.Server.Port),
		zap.String("address", fmt.Sprintf("http://localhost:%d", cfg.Server.Port)),
	)

	// 打印服务器信息到控制台
	s.printServerInfo(cfg.Server.Port)

	// 附加服务器监听和优雅关闭逻辑
	s.attachServerListenAndShutdown()

	return nil
}

// printServerInfo 在控制台打印服务器信息
func (s *Server) printServerInfo(port int) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✓ Server is running on http://localhost:%d\n", port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\nAvailable endpoints:")
	fmt.Printf("  • Main:    http://localhost:%d/?password=xxx&type=xxx\n", port)
	fmt.Printf("  • Health:  http://localhost:%d/health\n", port)
	fmt.Printf("  • Refresh: http://localhost:%d/refresh?password=xxx\n", port)
	fmt.Println("\nPress Ctrl+C to stop")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// attachServerListenAndShutdown 附加服务器监听和优雅关闭逻辑
func (s *Server) attachServerListenAndShutdown() {
	s.sc.Attach(func(done func(), closeSignal <-chan struct{}) {
		defer done()

		// 启动 HTTP 服务器（非阻塞）
		errChan := make(chan error, 1)
		go func() {
			errChan <- s.httpServer.ListenAndServe()
		}()

		// 等待服务器错误或关闭信号
		select {
		case err := <-errChan:
			// 服务器启动失败
			if err != nil && err != http.ErrServerClosed {
				s.logger.Error("Server error", zap.Error(err))
				s.sc.SendCloseSignal(err)
			}
		case <-closeSignal:
			// 收到关闭信号，执行优雅关闭
			s.gracefulShutdown()
		}
	})
}

// gracefulShutdown 优雅关闭服务器
func (s *Server) gracefulShutdown() {
	// 取消所有后台任务（通过 context）
	s.cancel()

	// 关闭 HTTP 服务器（5 秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Server shutdown error", zap.Error(err))
	} else {
		s.logger.Info("Server stopped gracefully")
	}

	// 刷新日志缓冲
	_ = s.logger.Sync()
}

// FetchResult 文件获取结果
type FetchResult struct {
	Name  string // 文件名称（node/template）
	Error error  // 获取过程中的错误
}

// fetchTask 文件获取任务
type fetchTask struct {
	name     string       // 任务名称
	fetchFn  func() error // 获取函数
	printMsg string       // 控制台打印信息
}

// performInitialFetch 执行首次文件获取
func (s *Server) performInitialFetch() error {
	s.logger.Info("Starting initial fetch of remote files")

	// 定义获取任务列表
	tasks := []fetchTask{
		{
			name:     "node",
			fetchFn:  fetcher.FetchNodeFile, // 获取节点文件
			printMsg: "Fetching node file...",
		},
		{
			name:     "template",
			fetchFn:  fetcher.FetchTemplateFile, // 获取模板文件
			printMsg: "Fetching template file...",
		},
	}

	// 并行获取所有文件
	results := s.fetchFilesParallel(tasks)

	// 统计获取结果
	var errors []error
	var successFiles []string

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("%s: %w", result.Name, result.Error))
		} else {
			successFiles = append(successFiles, result.Name)
		}
	}

	// 如果有错误，记录并返回
	if len(errors) > 0 {
		s.logger.Error("Initial fetch completed with errors",
			zap.Int("success_count", len(successFiles)),
			zap.Int("error_count", len(errors)),
			zap.Strings("success_files", successFiles),
			zap.Errors("errors", errors),
		)
		return fmt.Errorf("fetch errors: %v", errors)
	}

	s.logger.Info("Initial fetch completed successfully",
		zap.Int("files_fetched", len(successFiles)),
		zap.Strings("files", successFiles),
	)

	return nil
}

// fetchFilesParallel 并行获取多个文件
func (s *Server) fetchFilesParallel(tasks []fetchTask) []FetchResult {
	var wg sync.WaitGroup
	results := make([]FetchResult, len(tasks))

	// 为每个任务启动一个 goroutine
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t struct {
			name     string
			fetchFn  func() error
			printMsg string
		}) {
			defer wg.Done()

			fmt.Printf("  %s\n", t.printMsg)
			s.logger.Debug("Fetching file", zap.String("file", t.name))

			// 执行获取
			err := t.fetchFn()
			results[idx] = FetchResult{
				Name:  t.name,
				Error: err,
			}

			// 记录结果
			if err != nil {
				s.logger.Error("Failed to fetch file",
					zap.String("file", t.name),
					zap.Error(err),
				)
				fmt.Printf("  ✗ %s file failed: %v\n", t.name, err)
			} else {
				s.logger.Info("File fetched successfully", zap.String("file", t.name))
				fmt.Printf("  ✓ %s file fetched\n", t.name)
			}
		}(i, task)
	}

	// 等待所有任务完成
	wg.Wait()
	return results
}

// startAutoUpdate 启动定期自动更新服务
func (s *Server) startAutoUpdate(cfg *global.Config) {
	// 创建定时器
	ticker := time.NewTicker(cfg.GetRefreshInterval())
	defer ticker.Stop()

	s.logger.Info("✓ Auto-update service started",
		zap.Duration("interval", cfg.GetRefreshInterval()),
	)

	updateCount := 0

	for {
		select {
		case <-s.ctx.Done():
			// 收到停止信号，退出自动更新
			s.logger.Info("Auto-update service stopped",
				zap.Int("total_updates", updateCount),
			)
			return

		case <-ticker.C:
			// 定时器触发，执行自动更新
			updateCount++
			s.performAutoUpdate(updateCount, cfg)
		}
	}
}

// performAutoUpdate 执行自动更新
func (s *Server) performAutoUpdate(updateNum int, cfg *global.Config) {
	s.logger.Info("Auto-update triggered", zap.Int("update_number", updateNum))
	fmt.Printf("\n[%s] Auto-updating files (#%d)...\n",
		time.Now().Format("2006-01-02 15:04:05"), updateNum)

	// 定义更新任务（同首次获取）
	tasks := []fetchTask{
		{
			name:     "node",
			fetchFn:  fetcher.FetchNodeFile,
			printMsg: "Updating node file...",
		},
		{
			name:     "template",
			fetchFn:  fetcher.FetchTemplateFile,
			printMsg: "Updating template file...",
		},
	}

	// 并行获取文件
	results := s.fetchFilesParallel(tasks)

	// 检查更新结果
	hasError := false
	for _, result := range results {
		if result.Error != nil {
			hasError = true
			break
		}
	}

	// 记录更新结果
	if hasError {
		s.logger.Warn("Auto-update completed with errors",
			zap.Int("update_number", updateNum),
		)
		fmt.Printf("⚠ Auto-update #%d completed with errors\n", updateNum)
	} else {
		s.logger.Info("Auto-update completed successfully",
			zap.Int("update_number", updateNum),
		)
		fmt.Printf("✓ Auto-update #%d completed\n", updateNum)
	}

	// 计算并显示下次更新时间
	nextUpdate := time.Now().Add(cfg.GetRefreshInterval())
	s.logger.Info("Next update scheduled",
		zap.Time("next_update", nextUpdate),
		zap.Duration("interval", cfg.GetRefreshInterval()),
	)
	fmt.Printf("Next update: %s\n", nextUpdate.Format("2006-01-02 15:04:05"))
}
