# clashctl

Mihomo TUN 交互式 CLI 配置工具 — 输入订阅 URL，一键配置代理。

## 功能

- 🧙 交互式配置向导（TUI）
- 📡 输入机场订阅 URL 自动生成配置
- 🔀 TUN 模式 / mixed-port 模式
- ⚡ 一键启动 / 停止 / 重启 Mihomo
- 🔍 环境自检（9项检查）
- 📡 节点管理（查看 / 切换）
- 🔧 systemd 服务集成

## 快速安装

### 下载二进制

```bash
# 从 GitHub Releases 下载
curl -sL https://github.com/DUcotd/clashctl/releases/latest/download/clashctl-linux-amd64 -o clashctl
chmod +x clashctl
sudo mv clashctl /usr/local/bin/
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
# 交互式向导（推荐）
sudo clashctl init

# 环境自检
sudo clashctl doctor

# 命令式操作
clashctl export -u "https://你的订阅链接" -o /etc/mihomo/config.yaml
sudo clashctl start
clashctl status
clashctl nodes list
clashctl nodes use "节点名称"
```

## 命令列表

| 命令 | 说明 |
|------|------|
| `clashctl init` | 交互式配置向导 |
| `clashctl export` | 导出配置文件 |
| `clashctl start` | 启动 Mihomo |
| `clashctl stop` | 停止 Mihomo |
| `clashctl restart` | 重启 Mihomo |
| `clashctl status` | 查看运行状态 |
| `clashctl doctor` | 环境自检 |
| `clashctl nodes list` | 列出代理节点 |
| `clashctl nodes use` | 切换代理节点 |
| `clashctl nodes groups` | 列出代理组 |
| `clashctl config show` | 显示配置内容 |
| `clashctl config path` | 显示配置路径 |

## 前提条件

- Linux (systemd 发行版，Ubuntu 22.04+ / Debian 12+)
- [Mihomo](https://github.com/MetaCubeX/mihomo/releases) 已安装
- TUN 模式需要 root 权限

## 文档

- [用户指南](docs/USER_GUIDE.md)
- [开发文档](docs/DEVELOPMENT.md)

## 技术栈

- Go + Cobra (CLI)
- Bubble Tea (TUI)
- yaml.v3 (配置序列化)

## 许可证

MIT
