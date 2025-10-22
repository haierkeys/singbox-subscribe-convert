# Singbox Subscribe Convert

A configuration server for sing-box that automatically fetches remote templates and node data, then serves generated configurations with periodic auto-updates.

## ✨ Features

- ✅ **YAML Configuration** - Easy configuration management
- ✅ **Auto-Update** - Automatic periodic updates of remote files
- ✅ **Remote Fetching** - Fetch templates and nodes from remote URLs
- ✅ **Local Caching** - Fast startup with local cache
- ✅ **File Watching** - Auto-reload on cache changes
- ✅ **Health Check** - `/health` endpoint for monitoring
- ✅ **Manual Refresh** - `/refresh` endpoint for manual updates
- ✅ **Environment Variables** - Override config with env vars
- ✅ **Graceful Shutdown** - Proper signal handling
- ✅ **Detailed Logging** - Both file and console logging

## 📦 Installation

```bash
go install github.com/haierkeys/singbox-subscribe-convert@latest
