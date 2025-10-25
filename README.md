# Singbox Subscribe Convert

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24.1-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/docker/pulls/haierkeys/singbox-subscribe-convert.svg)](https://hub.docker.com/r/haierkeys/singbox-subscribe-convert)


一个强大的 sing-box 订阅转换服务，支持多模板配置、自动更新、热重载等功能，通过对远端配置进行二次聚合转换，解决不同 sing-box 版本（如避免使用 P 核等）带来的配置文件碎片化问题，实现一个订阅地址适配多端多版本使用。推荐使用 [Sub-Store](https://github.com/sub-store-org/Sub-Store) 做初步节点聚合（需生成 singbox 配置）。

## 📖 目录

- [功能特性](#-功能特性)
- [快速开始](#-快速开始)
- [安装部署](#-安装部署)
- [配置说明](#-配置说明)
- [使用方法](#-使用方法)
- [API 接口](#-api-接口)
- [模板变量定义](#-模板变量定义)
- [多模板功能](#-多模板功能)
- [常见问题](#-常见问题)
- [贡献](#-贡献)
- [许可证](#-许可证)
- [相关链接](#-相关链接)
- [作者](#-作者)
- [致谢](#-致谢)

## ✨ 功能特性

### 核心功能
- 🎯 **多模板支持** - 支持配置多个模板，可根据不同场景动态切换
- 🔄 **自动更新** - 定时自动获取和更新节点及模板配置
- 🔥 **热重载** - 配置文件变更自动检测和重载，无需重启服务
- 🔐 **密码认证** - 内置密码认证机制，保护订阅安全
- 📊 **健康检查** - 提供健康检查接口，方便监控服务状态
- 🐳 **Docker 支持** - 提供完整的 Docker 部署方案
- 🚀 **高性能** - 并行处理、文件缓存，响应迅速

### 高级特性
- 🎨 **自定义过滤器** - 支持节点名称过滤和自定义渲染
- 📦 **智能缓存** - 本地缓存机制，离线也能正常服务
- 🔍 **文件监控** - 实时监控缓存文件变化并自动重载
- 📈 **详细日志** - 完善的日志系统，支持文件和控制台输出
- ⚡ **优雅关闭** - 支持信号处理和优雅退出
- 🌐 **跨平台** - 支持 Linux、macOS、Windows、Docker

## 🚀 快速开始

### 使用 Docker (推荐)

```bash
# 拉取镜像
docker pull haierkeys/singbox-subscribe-convert:latest

# 运行容器
docker run -d \
  --name singbox-subscribe-convert \
  -p 9000:9000 \
  -v /path/to/config:/singbox-subscribe-convert/config \
  -v /path/to/storage:/singbox-subscribe-convert/storage \
  haierkeys/singbox-subscribe-convert:latest
```

### 使用 Docker Compose
```yaml
# Docker-Compose.yaml
  singbox-subscribe-convert:
    image: haierkeys/singbox-subscribe-convert:latest
    container_name: sub-convert
    ports:
      - "7000:9000"
    volumes:
      - /data/singbox-subscribe-convert/storage/:/singbox-subscribe-convert/storage/
      - /data/singbox-subscribe-convert/config/:/singbox-subscribe-convert/config/
    networks:
      - app-network  # 与 image-api 在同一网络1
```
```bash
# 启动服务
docker-compose up -d
```

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/haierkeys/singbox-subscribe-convert.git
cd singbox-subscribe-convert

# 编译项目
go build -o singbox-subscribe-convert .

# 运行服务
./singbox-subscribe-convert run
```

## 📦 安装部署

### 系统要求

- Go 1.24.1 或更高版本（源码编译）
- Docker 20.10+ 和 Docker Compose 2.0+（Docker 部署）
- 至少 100MB 可用内存
- 至少 50MB 可用磁盘空间

### 预编译二进制

从 [Releases](https://github.com/haierkeys/singbox-subscribe-convert/releases) 页面下载适合您系统的预编译二进制文件。

**支持的平台：**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

```bash
# Linux / macOS
chmod +x sb-sub-c
./sb-sub-c run

# Windows
sb-sub-c.exe run
```

### 使用 Makefile 编译

```bash
# 编译所有平台
make build-all

# 编译特定平台
make build-linux-amd64    # Linux AMD64
make build-linux-arm64    # Linux ARM64
make build-macos-amd64    # macOS Intel
make build-macos-arm64    # macOS Apple Silicon
make build-windows-amd64  # Windows AMD64
```

## ⚙️ 配置说明

### 配置文件位置

程序会按以下优先级查找配置文件：
1. 命令行指定：`-c` 或 `--config` 参数
2. `config/config-dev.yaml`
3. `config.yaml`
4. `config/config.yaml`

### 基础配置示例

```yaml
# 服务器配置
server:
  port: 9000              # 监听端口
  read_timeout: 15        # 读取超时（秒）
  write_timeout: 15       # 写入超时（秒）
  idle_timeout: 60        # 空闲超时（秒）

# 认证配置
auth:
  password: "your_secure_password"  # 访问密码

# 节点订阅配置（新格式）
subscription:
  url: "https://your-subscription-url"  # 订阅地址
  timeout: 30                           # 请求超时（秒）
  refresh_interval: 2                   # 刷新间隔（分钟）

# 模板配置（新格式）
templates:
  default:
    url: "https://template-url/default.json"
    name: "默认配置"
    no_node: "🎯 全球直连"
    enabled: true

  gaming:
    url: "https://template-url/gaming.json"
    name: "游戏加速"
    no_node: "🎯 全球直连"
    enabled: true

# 默认模板
default_template: "default"

# 缓存配置
cache:
  directory: "./data/cache"
  node_file: "node.json"
  template_file: "template.json"

# 日志配置
logging:
  production: true               # 生产模式
  file: "./data/log/server.log"  # 日志文件
  level: "info"                  # 日志级别：debug, info, warn, error
  max_size: 10                   # 单个日志文件最大大小（MB）
  max_backups: 3                 # 保留的旧日志文件数
  max_age: 7                     # 日志文件保留天数
```

### 配置项说明

#### Server (服务器配置)
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `port` | int | 9000 | HTTP 服务监听端口 |
| `read_timeout` | int | 15 | 读取超时时间（秒） |
| `write_timeout` | int | 15 | 写入超时时间（秒） |
| `idle_timeout` | int | 60 | 连接空闲超时时间（秒） |

#### Auth (认证配置)
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `password` | string | 是 | API 访问密码 |

#### Subscription (订阅配置)
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 是 | 节点订阅地址 |
| `timeout` | int | 否 | 请求超时（秒），默认 30 |
| `refresh_interval` | int | 是 | 自动刷新间隔（分钟） |

#### Templates (模板配置)
每个模板包含以下字段：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 是 | 模板文件 URL |
| `name` | string | 是 | 模板显示名称 |
| `no_node` | string | 是 | 无节点时的默认显示 |
| `enabled` | bool | 是 | 是否启用该模板 |

#### Cache (缓存配置)
| 参数 | 类型 | 说明 |
|------|------|------|
| `directory` | string | 缓存目录路径 |
| `node_file` | string | 节点缓存文件名 |
| `template_file` | string | 模板缓存文件名（旧格式） |

#### Logging (日志配置)
| 参数 | 类型 | 说明 |
|------|------|------|
| `production` | bool | 是否生产模式（JSON 格式） |
| `file` | string | 日志文件路径 |
| `level` | string | 日志级别 |
| `max_size` | int | 单文件最大大小（MB） |
| `max_backups` | int | 保留的日志文件数 |
| `max_age` | int | 日志保留天数 |

## 📚 使用方法

### 命令行选项

```bash
# 查看帮助
./singbox-subscribe-convert --help

# 查看版本
./singbox-subscribe-convert version

# 运行服务（使用默认配置）
./singbox-subscribe-convert run

# 指定配置文件
./singbox-subscribe-convert run -c /path/to/config.yaml

# 指定工作目录
./singbox-subscribe-convert run -d /path/to/workdir

# 指定端口（会覆盖配置文件）
./singbox-subscribe-convert run -p 8080
```

### 环境变量

以下环境变量可以覆盖配置文件中的设置：

```bash
export SERVER_PORT=9000                    # 服务器端口
export PASSWORD="your_password"            # 认证密码
export SUBSCRIPTION_URL="sub_url"          # 订阅地址
export DEFAULT_TEMPLATE="default"          # 默认模板
export CACHE_DIR="./data/cache"            # 缓存目录
export REFRESH_INTERVAL=2                  # 刷新间隔（分钟）
```

## 🔌 API 接口

### 主接口 - 获取配置

**请求：**
```
GET /?password=<密码>&template=<模板ID>&type=<类型>
```

**参数：**
- `password` (必需): 认证密码
- `template` (可选): 模板 ID，不指定则使用默认模板
- `type` (可选): 自定义类型参数，传递给模板

**示例：**
```
# 使用默认模板
http://localhost:9000/?password=your_password

# 指定模板
http://localhost:9000/?password=your_password&template=gaming

# 带自定义参数
http://localhost:9000/?password=your_password&template=gaming&type=custom
```

**响应：**
```json
{
  "dns": {...},
  "inbounds": [...],
  "outbounds": [...],
  "route": {...}
}
```

### 健康检查

**请求：**
```
GET /health
```

**示例：**
```
http://localhost:9000/health
```

**响应：**
```json
{
  "status": "ok",
  "has_data": true,
  "has_template": true,
  "node_count": 10,
  "template_count": 3
}
```

**状态码：**
- `200 OK` - 服务正常
- `503 Service Unavailable` - 服务降级（数据或模板未加载）

### 手动刷新

**请求：**
```
GET /refresh?password=<密码>
```

**参数：**
- `password` (必需): 认证密码

**示例：**
```
http://localhost:9000/refresh?password=your_password
```


**响应成功：**
```json
{
  "status": "success",
  "message": "Files refreshed successfully",
  "node_count": 10,
  "template_count": 3
}
```

**响应失败：**
```json
{
  "status": "error",
  "errors": [
    "node file: fetch error",
    "template gaming: load error"
  ]
}
```
## 📝 模板变量定义

模板文件支持两个核心变量，用于动态插入节点数据和生成 sing-box 配置。

### 1️⃣ Nodes - 插入完整节点配置

**作用：** 将所有订阅节点的完整配置插入到模板中。

**使用方式：**
```json
{
  "outbounds": [
    { "tag": "🚀 节点选择", "type": "selector", "outbounds": ["..."] },
    { "tag": "🎯 全球直连", "type": "direct" },

    {{ Nodes }}
  ]
}
```

**效果：** 会在指定位置插入所有节点的详细配置（包括服务器地址、端口、加密方式等完整信息）。

---

### 2️⃣ NotesName - 筛选节点名称

**作用：** 根据关键词筛选节点名称，生成节点列表。

**基本语法：** `{{ "关键词" | NotesName }}`

#### 使用场景

**场景 1：获取所有节点**
```json
{
  "tag": "🐸 手动切换",
  "type": "selector",
  "outbounds": [ {{ "" | NotesName }} ]
}
```
> 空字符串表示不过滤，返回所有节点名称

**场景 2：筛选特定地区节点**
```json
{
  "tag": "🇭🇰 香港节点",
  "type": "selector",
  "outbounds": [ {{ "香港" | NotesName }} ]
}
```
> 只返回节点名包含"香港"的节点，如：`🇭🇰 香港 01`、`香港专线`

**场景 3：筛选多个地区（OR 逻辑）**
```json
{
  "tag": "🇭🇰 港新节点",
  "type": "selector",
  "outbounds": [ {{ "香港|新加坡" | NotesName }} ]
}
```
> 使用 `|` 分隔多个关键词，返回包含"香港"**或**"新加坡"的节点

**场景 4：配合自动测速**
```json
{
  "tag": "♻️ 🇭🇰 港新自动",
  "type": "urltest",
  "outbounds": [ {{ "香港|新加坡" | NotesName }} ],
  "url": "http://www.gstatic.com/generate_204",
  "interval": "10m",
  "tolerance": 50
}
```
> 从筛选出的节点中自动选择延迟最低的

---

### 📝 完整示例

```json
{
  "outbounds": [
    {
      "tag": "🚀 节点选择",
      "type": "selector",
      "outbounds": ["🐸 手动切换", "♻️ 自动选择", "🇭🇰 香港节点", "🇯🇵 日本节点", "🎯 全球直连"]
    },
    {
      "tag": "🐸 手动切换",
      "type": "selector",
      "outbounds": [ {{ "" | NotesName }} ]
    },
    {
      "tag": "♻️ 自动选择",
      "type": "urltest",
      "outbounds": [ {{ "" | NotesName }} ],
      "url": "http://www.gstatic.com/generate_204",
      "interval": "10m"
    },
    {
      "tag": "🇭🇰 香港节点",
      "type": "selector",
      "outbounds": [ {{ "香港" | NotesName }} ]
    },
    {
      "tag": "🇯🇵 日本节点",
      "type": "selector",
      "outbounds": [ {{ "日本" | NotesName }} ]
    },
    { "tag": "🎯 全球直连", "type": "direct" },

    {{ Nodes }}
  ]
}
```

---

### 💡 使用提示

- **关键词匹配**：支持节点名称的模糊匹配，例如 `"香港"` 可以匹配 `🇭🇰 香港 01`、`香港-IPLC` 等
- **多关键词**：使用 `|` 分隔，例如 `"香港|HK|Hong Kong"` 可以匹配多种命名方式
- **无匹配处理**：如果筛选后没有任何节点，会自动使用配置中的 `no_node` 值（如 `🎯 全球直连`）
- **区分大小写**：关键词匹配区分大小写，注意与实际节点名称保持一致

## 🎨 多模板功能

### 配置多个模板

```yaml
templates:
  # OpenWRT singbox1.12 配置
  default:
    url: "https://example.com/templates/default.json"
    name: "OpenWRT"
    no_node: "🎯 全球直连"
    enabled: true

  # IOS singbox1.10 配置
  ios:
    url: "https://example.com/templates/gaming.json"
    name: "IOS"
    no_node: "🎯 全球直连"
    enabled: true

default_template: "default"
```

### 使用不同模板

```bash
# 默认模板
curl "http://localhost:9000/?password=xxx"

# OPENWRT singbox1.12 配置
curl "http://localhost:9000/?password=xxx&template=default"

# IOS singbox1.10 配置
curl "http://localhost:9000/?password=xxx&template=ios"
```

### 模板特性

- ✅ **独立缓存** - 每个模板有独立的缓存文件
- ✅ **并行更新** - 多个模板同时更新，提高效率
- ✅ **动态加载** - 可通过配置启用/禁用模板
- ✅ **热重载** - 模板文件变更自动重新加载


## ❓ 常见问题

### 1. 服务启动失败

**问题：** 服务启动后立即退出

**解决方案：**
- 检查配置文件是否正确
- 检查端口是否被占用
- 查看日志文件 `data/log/server.log`

### 2. 无法获取节点

**问题：** 订阅地址无法访问

**解决方案：**
- 检查订阅 URL 是否正确
- 检查网络连接
- 检查防火墙设置
- 使用 `/refresh?password=xxx` 手动刷新

### 3. 模板未生效

**问题：** 请求返回 "Template not found"

**解决方案：**
- 确认模板 ID 在配置中存在
- 确认模板的 `enabled` 为 `true`
- 使用 `/health` 检查已加载的模板数量
- 检查模板 URL 是否可访问

### 4. 认证失败

**问题：** 返回 "Password Error"

**解决方案：**
- 检查 URL 参数中的 `password` 是否正确
- 检查配置文件中的 `auth.password` 设置
- 确保密码没有特殊字符需要 URL 编码

### 5. Docker 容器无法访问

**问题：** 容器运行但无法访问服务

**解决方案：**
- 检查端口映射是否正确
- 确认容器状态：`docker ps`
- 查看容器日志：`docker logs singbox-subscribe-convert`
- 检查防火墙和网络设置

### 6. 更新不生效

**问题：** 修改配置后未生效

**解决方案：**
- 配置文件变更会自动重载（需等待几秒）
- 或手动重启服务
- 检查配置文件语法是否正确

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 [Apache License 2.0](LICENSE) 许可证。

## 🔗 相关链接

- [sing-box 官方文档](https://sing-box.sagernet.org/)
- [Docker Hub](https://hub.docker.com/r/haierkeys/singbox-subscribe-convert)
- [GitHub Issues](https://github.com/haierkeys/singbox-subscribe-convert/issues)
- [Sub-Store](https://github.com/sub-store-org/Sub-Store)

## 👤 作者

**HaierKeys**

- Email: haierkeys@gmail.com
- GitHub: [@haierkeys](https://github.com/haierkeys)

## 🙏 致谢

感谢所有为本项目做出贡献的开发者！

---

如有问题或建议，欢迎提交 [Issue](https://github.com/haierkeys/singbox-subscribe-convert/issues)。
