# clashctl

[![Go Version](https://img.shields.io/github/go-mod/go-version/DUcotd/clashctl?label=Go&color=00ADD8)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/DUcotd/clashctl?label=Release&color=blue)](https://github.com/DUcotd/clashctl/releases)
[![License](https://img.shields.io/github/license/DUcotd/clashctl?color=green)](LICENSE)
[![Stars](https://img.shields.io/github/stars/DUcotd/clashctl?style=flat)](https://github.com/DUcotd/clashctl/stargazers)

Mihomo 交互式 CLI 配置工具 — 输入订阅链接，一键配置代理。

## 安装

### 一键安装

```bash
curl -sLO https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh
chmod +x install.sh
sudo ./install.sh
sudo clashctl init
```

### 手动安装

```bash
sudo curl -sL https://github.com/DUcotd/clashctl/releases/latest/download/clashctl-linux-amd64 -o /usr/local/bin/clashctl
sudo chmod +x /usr/local/bin/clashctl
sudo clashctl service install
```

### 编译安装

```bash
git clone https://github.com/DUcotd/clashctl.git
cd clashctl
go build -o clashctl .
sudo mv clashctl /usr/local/bin/
```

## 使用

### 交互式向导

```bash
sudo clashctl init
```

引导流程：订阅来源 → 运行模式 → 高级参数 → 配置预览 → 安装 → 节点管理

### 节点管理

```bash
clashctl nodes
```

| 按键 | 功能 |
|------|------|
| `j`/`k` 或 `↑`/`↓` | 选择 |
| `Enter` | 进入组 / 切换节点 |
| `t` | 延迟测试 |
| `s` | 切换排序（默认/延迟/名称/协议） |
| `/` | 搜索 |
| `i` | 节点详情 |
| `c` | 复制节点名 |
| `?` | 快捷键帮助 |
| `r` | 刷新 |
| `Esc` | 返回 |
| `q` | 退出 |

### 常用命令

```bash
clashctl doctor                        # 环境自检
clashctl doctor openai                 # OpenAI/Codex 诊断

clashctl nodes list --json             # 列出节点
clashctl nodes groups --json           # 列出代理组
clashctl nodes use "节点名"            # 切换节点
clashctl nodes test --all-groups       # 全组测速

sudo clashctl service install          # 安装服务
sudo clashctl service start|stop|restart
clashctl service status --json

sudo clashctl config export -u "订阅链接" -o /etc/mihomo/config.yaml
clashctl config import -f sub.txt --apply --start
clashctl config show --json

clashctl backup                        # 备份配置
clashctl backup restore                # 恢复备份

clashctl update --dry-run              # 检查更新
clashctl self                          # 同 update
```

## 命令总览

| 命令 | 说明 |
|------|------|
| `init` | 交互式配置向导 |
| `nodes` | 节点管理 TUI |
| `nodes list` | 列出节点 (`--json`) |
| `nodes groups` | 列出代理组 (`--json`) |
| `nodes use <name>` | 切换节点 |
| `nodes test` | 延迟测试 |
| `service install` | 安装 Mihomo 服务 |
| `service start/stop/restart` | 服务控制 |
| `service status` | 服务状态 (`--json`) |
| `doctor` | 环境自检 (8 项, `--tun` 时 11 项) |
| `doctor openai` | OpenAI/Codex 诊断 |
| `config export` | 导出配置 (`--json`) |
| `config import` | 本地文件导入 |
| `config show` | 查看配置 (`--json`) |
| `config path` | 配置路径 |
| `backup` | 备份配置 (`--json`) |
| `backup restore` | 恢复备份 (`--json`) |
| `update` / `self` | 自更新 |
| `version` | 版本信息 |

## 配置路径

| 文件 | 路径 |
|------|------|
| Mihomo 配置 | `/etc/mihomo/config.yaml` |
| Provider 缓存 | `/etc/mihomo/providers/` |
| clashctl 配置 | `~/.config/clashctl/config.yaml` |
| 备份 | `~/.config/clashctl/backup/` |
| systemd 服务 | `/etc/systemd/system/clashctl-mihomo.service` |

## 安全

- 下载的二进制和校验文件默认仅信任 GitHub 官方 Release；设置 `CLASHCTL_ALLOW_UNTRUSTED_MIRROR=1` 可启用第三方镜像兜底
- `controller_addr` 仅允许回环地址，防止 API 暴露到外部网络
- 订阅 URL 下载使用 DNS 级别 SSRF 防护，拒绝内网/本地地址
- 配置文件读写强制路径校验，防止目录遍历攻击
- 远程订阅自动屏蔽 `script`、`dns`、`tun` 等高风险字段

## 前提

- Linux (systemd, Ubuntu 22.04+ / Debian 12+ / Fedora)
- TUN 模式需要 root 或 `CAP_NET_ADMIN`
- Mihomo 首次使用时自动下载安装

## 文档

- [用户指南](docs/USER_GUIDE.md)
- [开发文档](docs/DEVELOPMENT.md)

## 技术栈

Go + [Cobra](https://github.com/spf13/cobra) + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [yaml.v3](https://github.com/go-yaml/yaml)

## 许可证

MIT
