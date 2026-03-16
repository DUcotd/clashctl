# myproxy

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
curl -sL https://github.com/<owner>/myproxy/releases/latest/download/myproxy-linux-amd64 -o myproxy
chmod +x myproxy
sudo mv myproxy /usr/local/bin/
```

### 从源码编译

```bash
# 需要 Go 1.21+
git clone https://github.com/<owner>/myproxy.git
cd myproxy
export GOPROXY=https://goproxy.cn,direct
go build -o myproxy .
sudo mv myproxy /usr/local/bin/
```

## 使用

```bash
# 交互式向导（推荐）
sudo myproxy init

# 环境自检
sudo myproxy doctor

# 命令式操作
myproxy export -u "https://你的订阅链接" -o /etc/mihomo/config.yaml
sudo myproxy start
myproxy status
myproxy nodes list
myproxy nodes use "节点名称"
```

## 命令列表

| 命令 | 说明 |
|------|------|
| `myproxy init` | 交互式配置向导 |
| `myproxy export` | 导出配置文件 |
| `myproxy start` | 启动 Mihomo |
| `myproxy stop` | 停止 Mihomo |
| `myproxy restart` | 重启 Mihomo |
| `myproxy status` | 查看运行状态 |
| `myproxy doctor` | 环境自检 |
| `myproxy nodes list` | 列出代理节点 |
| `myproxy nodes use` | 切换代理节点 |
| `myproxy nodes groups` | 列出代理组 |
| `myproxy config show` | 显示配置内容 |
| `myproxy config path` | 显示配置路径 |

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
