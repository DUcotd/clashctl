# clashctl

Mihomo 交互式 CLI 配置工具 — 输入订阅 URL，一键配置代理。

## 功能

- 🧙 **交互式配置向导**（TUI）— 输入订阅链接，一键完成全部配置
- 📡 **自动下载 Mihomo** — 内核自动安装，无需手动下载
- 🔀 TUN 模式 / mixed-port 模式
- ⚡ 一键启动 / 停止 / 重启 Mihomo
- 🔍 环境自检（默认 8 项，`--tun` 时 11 项）
- 📡 **节点管理**（延迟测试 / 切换 / 刷新）
- 🔧 systemd 服务集成
- 🔄 自动更新

## 快速安装（推荐）

```bash
# 下载安装脚本（推荐，支持校验）
curl -sLO https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh
# 可选：校验脚本完整性
# sha256sum install.sh
chmod +x install.sh
sudo ./install.sh

# 启动配置向导
sudo clashctl init
```

两步搞定，不需要手动下载 Mihomo。

> ⚠️ 安全提示：避免使用 `curl ... | sudo bash` 管道方式，建议先下载脚本再执行。

### 手动安装

```bash
# 只安装 clashctl
sudo curl -sL https://github.com/DUcotd/clashctl/releases/latest/download/clashctl-linux-amd64 -o /usr/local/bin/clashctl
sudo chmod +x /usr/local/bin/clashctl

# 安装 Mihomo 内核（自动下载最新版）
sudo clashctl service install
```

### 从源码编译

```bash
# 需要 Go 1.21+
git clone https://github.com/DUcotd/clashctl.git
cd clashctl
export GOPROXY=https://goproxy.cn,direct
go build -o clashctl .
sudo mv clashctl /usr/local/bin/
```

## 使用

```bash
# 交互式向导（推荐，全流程）
sudo clashctl init

# 环境自检
sudo clashctl doctor
clashctl doctor --json
clashctl doctor openai

# 日常操作
clashctl nodes
clashctl nodes list --json
clashctl nodes groups --json
clashctl nodes use "节点名" --json
clashctl service status
clashctl service status --json
clashctl nodes test
clashctl nodes test --json
clashctl nodes test --all-groups
clashctl backup
clashctl backup --json
clashctl backup restore --json
clashctl backup restore

# 配置与脚本化操作
sudo clashctl service install --json
sudo clashctl config export -u "https://你的订阅链接" -o /etc/mihomo/config.yaml
clashctl config export -u "https://你的订阅链接" -o config.yaml --json
clashctl config import -f sub.txt -o config.yaml
clashctl config import -f sub.txt --apply --start
clashctl config import -f sub.txt --json
clashctl config path --json
clashctl config show --json
sudo clashctl service start
clashctl update --dry-run
clashctl update --json --dry-run
```

## TUI 节点管理

在 `clashctl init` 向导完成后，如果 Mihomo 已运行且 Controller API 可达，结果页会提示你按 `Enter` 或 `n` 进入节点管理界面：

| 按键 | 功能 |
|------|------|
| `↑`/`↓` 或 `j`/`k` | 上下选择 |
| `Enter` | 进入组 / 切换节点 |
| `t` | 测试所有节点延迟 |
| `r` | 刷新列表 |
| `Esc` | 返回上一级 |

延迟显示：`✨ <100ms` → `100-300ms` → `⚠️ 300-1000ms` → `🔴 >1s` → `超时`

也可以直接跳过向导进入节点管理：

```bash
clashctl nodes
```

## 命令列表

| 命令 | 说明 |
|------|------|
| `clashctl init` | 交互式配置向导（含节点管理） |
| `clashctl nodes` | 默认进入节点测速与切换 TUI |
| `clashctl service ...` | 安装 / 启动 / 停止 / 重启 / 查看状态 |
| `clashctl doctor` | 环境自检（默认 8 项，`--tun` 时 11 项） |
| `clashctl doctor openai` | 诊断 OpenAI/Codex 登录链路（含 `chatgpt.com/backend-api`） |
| `clashctl service install` | 安装 Mihomo 内核，支持 `--json` |
| `clashctl config export` | 导出配置文件，支持 `--json` |
| `clashctl config import` | 从本地原始订阅文件生成静态配置，可直接应用并启动，支持 `--json` |
| `clashctl config show` | 显示配置内容，支持 `--json` |
| `clashctl config path` | 显示配置路径，支持 `--json` |
| `clashctl nodes list` | 列出代理节点，支持 `--json` |
| `clashctl nodes test` | 测试代理组节点延迟 |
| `clashctl nodes use` | 切换代理节点，支持 `--json` |
| `clashctl nodes groups` | 列出代理组，支持 `--json` |
| `clashctl backup` | 备份当前 Mihomo 与 clashctl 配置，支持 `--json` |
| `clashctl backup restore [备份文件名]` | 列出或恢复历史备份，支持 `--json` |
| `clashctl update` | 检查并更新 clashctl，支持 `--json` |
| `clashctl self` | `clashctl update` 的兼容别名 |
| `clashctl version` | 显示版本信息 |

## 前提条件

- Linux (systemd 发行版，Ubuntu 22.04+ / Debian 12+)
- TUN 模式需要 root 权限
- Mihomo 会在首次使用时自动下载，也可手动安装
- `init` 默认优先将订阅转成更适合服务器使用的静态配置，尽量避免服务器再次直连拉取 provider
- `mixed-port` 模式会在 `init` 成功后自动写入当前 shell 的代理环境配置（含 `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` / `NODE_USE_ENV_PROXY=1`）；新开终端自动生效，当前终端需手动 `source` 一次

## 配置与备份路径

| 文件 | 路径 |
|------|------|
| Mihomo 配置 | `/etc/mihomo/config.yaml`（默认，可在向导中修改） |
| Provider 缓存 | `/etc/mihomo/providers/airport.yaml`（默认） |
| clashctl 自身配置 | `~/.config/clashctl/config.yaml` |
| 备份目录 | `~/.config/clashctl/backup/` |
| systemd 服务 | `/etc/systemd/system/clashctl-mihomo.service` |

## 文档

- [用户指南](docs/USER_GUIDE.md)
- [开发文档](docs/DEVELOPMENT.md)

## 技术栈

- Go + Cobra (CLI)
- Bubble Tea (TUI)
- yaml.v3 (配置序列化)

## 许可证

MIT
