# clashctl 用户指南

## 简介

clashctl 是一个运行在 Linux 终端的交互式工具，帮助你通过输入订阅 URL 快速完成 Mihomo 代理的配置、启动与管理。

## 前提条件

- **root 权限**（TUN 模式必需）
- **systemd**（推荐，用于服务管理）
- **Mihomo** 可选预装；若未安装，`clashctl init` 会自动下载

## 快速开始

```bash
sudo clashctl init
```

按向导提示操作即可。

## 命令参考

### clashctl init
启动交互式配置向导。

默认只需要输入订阅 URL 或本地订阅文件路径。`init` 会优先把订阅转成更适合服务器使用的静态配置，尽量减少 provider 拉取失败带来的问题。

如果选择 `mixed-port` 模式，向导会在完成后自动把 `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` 写入当前用户的 shell 配置文件；新开终端自动生效，当前终端执行一次 `source ~/.bashrc`（或对应 shell 配置文件）即可。

### clashctl advanced export
导出 Mihomo 配置文件。

```bash
clashctl advanced export -u <订阅URL> [-m tun|mixed] [-p 端口] [-o 输出路径]
```

### clashctl advanced import
从本地原始订阅文件生成静态 Mihomo 配置，适用于服务器无法直接拉取订阅 URL 的场景。

```bash
clashctl advanced import -f sub.txt -o config.yaml
clashctl advanced import -f sub.txt --apply --start
cat sub.txt | clashctl advanced import -f - --apply --start
```

### clashctl service start / stop / restart
管理 Mihomo 服务。

### clashctl service status
查看运行状态、配置目录、Controller API、代理组和当前节点。

### clashctl doctor
环境自检（默认 8 项；启用 `--tun` 时为 11 项检查）。

```bash
clashctl doctor         # 默认检查常规环境
clashctl doctor --tun   # 额外检查 TUN 相关条件
clashctl doctor openai  # 诊断 OpenAI/Codex 登录链路（含 chatgpt.com/backend-api）
```

### clashctl nodes
节点管理。

```bash
clashctl nodes                      # 默认进入节点管理 TUI
clashctl nodes list [组名]           # 列出节点
clashctl nodes test [组名]           # 测试一个代理组的全部节点延迟
clashctl nodes test --all-groups     # 测试所有代理组
clashctl nodes use "节点名" [组名]    # 切换节点
clashctl nodes groups               # 列出所有代理组
```

### clashctl advanced config
配置查看。

```bash
clashctl advanced config show    # 显示配置内容
clashctl advanced config path    # 显示配置路径
```

## 配置文件

| 文件 | 路径 |
|------|------|
| Mihomo 配置 | `/etc/mihomo/config.yaml`（默认，可在向导中修改） |
| Provider 缓存 | `/etc/mihomo/providers/airport.yaml`（默认） |
| systemd 服务 | `/etc/systemd/system/clashctl-mihomo.service` |

## 常见问题

### 未找到 mihomo
运行 `sudo clashctl advanced install`，或直接执行 `sudo clashctl init` 让工具自动下载安装。

### TUN 模式需要 root
使用 `sudo clashctl init` 或 `sudo clashctl service start`。

### 端口被占用
检查是否有其他 Mihomo 实例在运行：`ps aux | grep mihomo`

### 订阅 URL 无法访问
检查网络连接，确认 URL 以 http/https 开头。

### 服务器能启动 Mihomo，但节点没有加载出来
这通常说明 Controller API 已启动，但服务器无法直连订阅 URL，或订阅返回的是原始节点链接而非 YAML。

可先在本地下载订阅，再执行：

```bash
base64 -d sub.txt > links.txt   # 如果文件是 base64
clashctl advanced import -f links.txt -o config.yaml
clashctl advanced import -f links.txt --apply --start
```
