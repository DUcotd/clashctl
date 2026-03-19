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
# 一键安装 clashctl + Mihomo 内核
curl -sL https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh | sudo bash

# 启动配置向导
sudo clashctl init
```

两步搞定，不需要手动下载 Mihomo。

### 手动安装

```bash
# 只安装 clashctl
sudo curl -sL https://github.com/DUcotd/clashctl/releases/latest/download/clashctl-linux-amd64 -o /usr/local/bin/clashctl
sudo chmod +x /usr/local/bin/clashctl

# 安装 Mihomo 内核（自动下载最新版）
sudo clashctl install
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

# 安装 Mihomo 内核
sudo clashctl install

# 环境自检
sudo clashctl doctor

# 命令式操作
sudo clashctl export -u "https://你的订阅链接" -o /etc/mihomo/config.yaml
clashctl import -f sub.txt -o config.yaml
clashctl import -f sub.txt --apply --start
sudo clashctl start
clashctl status
clashctl nodes list
clashctl nodes use "节点名称"
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

## 命令列表

| 命令 | 说明 |
|------|------|
| `clashctl init` | 交互式配置向导（含节点管理） |
| `clashctl install` | 安装 Mihomo 内核 |
| `clashctl export` | 导出配置文件 |
| `clashctl import` | 从本地原始订阅文件生成静态配置，可直接应用并启动 |
| `clashctl start` | 启动 Mihomo |
| `clashctl stop` | 停止 Mihomo |
| `clashctl restart` | 重启 Mihomo |
| `clashctl status` | 查看运行状态 |
| `clashctl doctor` | 环境自检（默认 8 项，`--tun` 时 11 项） |
| `clashctl nodes list` | 列出代理节点 |
| `clashctl nodes use` | 切换代理节点 |
| `clashctl nodes groups` | 列出代理组 |
| `clashctl config show` | 显示配置内容 |
| `clashctl config path` | 显示配置路径 |
| `clashctl update` | 检查并更新 clashctl |
| `clashctl version` | 显示版本信息 |

## 前提条件

- Linux (systemd 发行版，Ubuntu 22.04+ / Debian 12+)
- TUN 模式需要 root 权限
- Mihomo 会在首次使用时自动下载，也可手动安装
- `mixed-port` 模式只提供本地代理端口，服务器流量需要显式配置代理环境变量

## 文档

- [用户指南](docs/USER_GUIDE.md)
- [开发文档](docs/DEVELOPMENT.md)

## 技术栈

- Go + Cobra (CLI)
- Bubble Tea (TUI)
- yaml.v3 (配置序列化)

## 许可证

MIT
