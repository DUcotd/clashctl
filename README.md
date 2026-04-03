# clashctl

[![Go Version](https://img.shields.io/github/go-mod/go-version/DUcotd/clashctl?label=Go&color=00ADD8)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/DUcotd/clashctl?label=Release&color=blue)](https://github.com/DUcotd/clashctl/releases)
[![License](https://img.shields.io/github/license/DUcotd/clashctl?color=green)](LICENSE)
[![Stars](https://img.shields.io/github/stars/DUcotd/clashctl?style=flat)](https://github.com/DUcotd/clashctl/stargazers)

Mihomo 交互式 CLI 配置工具 — 输入订阅 URL，一键配置代理。

## 功能

- 🧙 **交互式配置向导（TUI）** — 输入订阅链接，一键完成全部配置
- 📡 **自动下载 Mihomo** — 内核自动安装，无需手动下载
- 🔀 **TUN 模式 / mixed-port 模式** — 灵活切换
- 📡 **节点管理 TUI** — 搜索、排序、延迟测试、一键切换
- 🔍 **环境自检** — 默认 8 项检查，`--tun` 时 11 项
- 🔧 **systemd 服务集成** — 一键安装/启动/停止/重启
- 🔄 **自动更新** — 安全校验，只信任官方 Release
- 💾 **备份/恢复** — 自动备份配置，随时回滚

## 快速安装

### 一键安装（推荐）

```bash
curl -sLO https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh
chmod +x install.sh
sudo ./install.sh
sudo clashctl init
```

两步搞定，不需要手动下载 Mihomo。

> ⚠️ 安全提示：避免使用 `curl ... | sudo bash` 管道方式，建议先下载脚本再执行。

### 手动安装

```bash
sudo curl -sL https://github.com/DUcotd/clashctl/releases/latest/download/clashctl-linux-amd64 -o /usr/local/bin/clashctl
sudo chmod +x /usr/local/bin/clashctl
sudo clashctl service install
```

### 从源码编译

```bash
git clone https://github.com/DUcotd/clashctl.git
cd clashctl
export GOPROXY=https://goproxy.cn,direct
go build -o clashctl .
sudo mv clashctl /usr/local/bin/
```

## 使用

### 交互式向导（推荐）

```bash
sudo clashctl init
```

向导引导你完成：订阅来源选择 → 运行模式 → 高级参数 → 配置预览 → 执行安装 → 节点管理

### 节点管理 TUI

```bash
clashctl nodes
```

| 按键 | 功能 |
|------|------|
| `↑`/`↓` 或 `j`/`k` | 上下选择 |
| `Enter` | 进入组 / 切换节点（需确认） |
| `t` | 测试所有节点延迟 |
| `s` | 切换排序：默认 / 延迟 / 名称 / 协议 |
| `/` | 搜索节点 / 代理组 |
| `i` | 查看节点详情 |
| `g` / `*` | 跳到当前选中节点 |
| `c` | 复制节点名到剪贴板 |
| `?` | 显示快捷键帮助 |
| `r` | 刷新列表 |
| `Esc` | 返回上一级 |
| `q` | 退出 |

延迟显示规则：`<100ms` → `100-300ms` → `300-1000ms` → `>1s` → 未测试

### 常用命令

```bash
# 环境自检
sudo clashctl doctor
clashctl doctor --json
clashctl doctor openai          # 诊断 OpenAI/Codex 登录链路

# 节点操作
clashctl nodes list --json      # 列出所有节点
clashctl nodes groups --json    # 列出代理组
clashctl nodes use "节点名"     # 切换节点
clashctl nodes test             # 测速
clashctl nodes test --all-groups

# 服务管理
clashctl service status --json
sudo clashctl service install
sudo clashctl service start/stop/restart

# 配置管理
sudo clashctl config export -u "https://订阅链接" -o /etc/mihomo/config.yaml
clashctl config import -f sub.txt --apply --start
clashctl config show --json
clashctl config path

# 备份
clashctl backup
clashctl backup restore
clashctl backup restore "备份文件名"

# 更新
clashctl update --dry-run
clashctl self                   # update 的别名
```

## 命令总览

| 命令 | 说明 |
|------|------|
| `clashctl init` | 交互式配置向导（含节点管理） |
| `clashctl nodes` | 节点管理 TUI（测速/切换/搜索/排序） |
| `clashctl nodes list` | 列出代理节点，支持 `--json` |
| `clashctl nodes groups` | 列出代理组，支持 `--json` |
| `clashctl nodes use` | 切换代理节点，支持 `--json` |
| `clashctl nodes test` | 测试代理组节点延迟 |
| `clashctl service install` | 安装 Mihomo 内核 |
| `clashctl service start/stop/restart` | 服务控制 |
| `clashctl service status` | 查看服务状态，支持 `--json` |
| `clashctl doctor` | 环境自检（8 项，`--tun` 时 11 项） |
| `clashctl doctor openai` | 诊断 OpenAI/Codex 登录链路 |
| `clashctl config export` | 导出配置文件，支持 `--json` |
| `clashctl config import` | 从本地文件生成静态配置 |
| `clashctl config show` | 显示配置内容，支持 `--json` |
| `clashctl config path` | 显示配置路径，支持 `--json` |
| `clashctl backup` | 备份配置，支持 `--json` |
| `clashctl backup restore` | 恢复备份，支持 `--json` |
| `clashctl update` / `self` | 检查并更新 clashctl |
| `clashctl version` | 显示版本信息 |

## 前提条件

- **Linux**（systemd 发行版，Ubuntu 22.04+ / Debian 12+）
- TUN 模式需要 `CAP_NET_ADMIN` 权限（root 或 setcap）
- Mihomo 会在首次使用时自动下载
- `init` 默认优先将订阅转成静态配置，避免服务器再次直连拉取 provider
- 远程 URL 订阅如果是 provider-only 配置，会直接拒绝；请先在本地展开成静态节点或改用 `clashctl config import`
- `controller_addr` 仅支持本地回环地址（`127.0.0.1` / `localhost` / `[::1]`）
- `mixed-port` 模式会在 `init` 成功后自动写入代理环境配置（`HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` / `NODE_USE_ENV_PROXY=1`）

## 安全说明

- `clashctl update` 和 `clashctl service install` 默认只信任 GitHub 官方 Release 元数据
- 如确需接受第三方镜像兜底，必须显式设置 `CLASHCTL_ALLOW_UNTRUSTED_MIRROR=1`
- 该环境变量会放宽供应链信任边界，仅建议在你明确接受风险时使用

## 配置路径

| 文件 | 默认路径 |
|------|----------|
| Mihomo 配置 | `/etc/mihomo/config.yaml` |
| Provider 缓存 | `/etc/mihomo/providers/airport.yaml` |
| clashctl 配置 | `~/.config/clashctl/config.yaml` |
| 备份目录 | `~/.config/clashctl/backup/` |
| systemd 服务 | `/etc/systemd/system/clashctl-mihomo.service` |

## 文档

- [📖 用户指南](docs/USER_GUIDE.md)
- [🛠️ 开发文档](docs/DEVELOPMENT.md)

## 技术栈

- **Go 1.21** + [Cobra](https://github.com/spf13/cobra) (CLI)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) (TUI)
- [yaml.v3](https://github.com/go-yaml/yaml) (配置序列化)

## 许可证

MIT
