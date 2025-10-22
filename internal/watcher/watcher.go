package watcher

import (
	"context"
	"path/filepath"
	"time"

	"github.com/haierkeys/singbox-subscribe-convert/global"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// Start 启动文件监控
func Start(ctx context.Context, cfg *global.Config, logger *zap.Logger, onNodeChange, onTemplateChange func() error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Error creating watcher",
			zap.Error(err),
		)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(cfg.Cache.Directory); err != nil {
		logger.Error("Error adding cache directory to watcher",
			zap.Error(err),
			zap.String("directory", cfg.Cache.Directory),
		)
		return
	}

	logger.Info("✓ File watcher started",
		zap.String("monitoring", cfg.Cache.Directory),
	)

	debounce := make(map[string]time.Time)
	debounceInterval := 1 * time.Second

	nodeFilePath, _ := filepath.Abs(cfg.GetNodeFilePath())
	templateFilePath, _ := filepath.Abs(cfg.GetTemplateFilePath())

	for {
		select {
		case <-ctx.Done():
			logger.Info("File watcher stopped")
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				absPath, _ := filepath.Abs(event.Name)

				if lastTime, exists := debounce[absPath]; exists {
					if time.Since(lastTime) < debounceInterval {
						continue
					}
				}
				debounce[absPath] = time.Now()

				if absPath == nodeFilePath {
					logger.Info("Node file changed, reloading...",
						zap.String("file", absPath),
					)
					if err := onNodeChange(); err != nil {
						logger.Error("Error reloading node data",
							zap.Error(err),
						)
					} else {
						logger.Info("✓ Node data reloaded successfully")
					}
				} else if absPath == templateFilePath {
					logger.Info("Template file changed, reloading...",
						zap.String("file", absPath),
					)
					if err := onTemplateChange(); err != nil {
						logger.Error("Error reloading template",
							zap.Error(err),
						)
					} else {
						logger.Info("✓ Template reloaded successfully")
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error("Watcher error",
				zap.Error(err),
			)
		}
	}
}
