# clashctl 用户指南

## 简介

clashctl 是一个运行在 Linux 终端的交互式工具，帮助你通过输入机场订阅 URL 快速完成 Mihomo 代理的配置、启动与管理。

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

### clashctl export
导出 Mihomo 配置文件。

```bash
clashctl export -u <订阅URL> [-m tun|mixed] [-p 端口] [-o 输出路径]
```

### clashctl start / stop / restart
管理 Mihomo 服务。

### clashctl status
查看运行状态、配置目录、Controller API、代理组和当前节点。

### clashctl doctor
环境自检（默认 8 项；启用 `--tun` 时为 11 项检查）。

```bash
clashctl doctor         # 默认检查常规环境
clashctl doctor --tun   # 额外检查 TUN 相关条件
```

### clashctl nodes
节点管理。

```bash
clashctl nodes list [组名]           # 列出节点
clashctl nodes use "节点名" [组名]    # 切换节点
clashctl nodes groups               # 列出所有代理组
```

### clashctl config
配置查看。

```bash
clashctl config show    # 显示配置内容
clashctl config path    # 显示配置路径
```

## 配置文件

| 文件 | 路径 |
|------|------|
| Mihomo 配置 | `/etc/mihomo/config.yaml`（默认，可在向导中修改） |
| Provider 缓存 | `/etc/mihomo/providers/airport.yaml`（默认） |
| systemd 服务 | `/etc/systemd/system/clashctl-mihomo.service` |

## 常见问题

### 未找到 mihomo
运行 `sudo clashctl install`，或直接执行 `sudo clashctl init` 让工具自动下载安装。

### TUN 模式需要 root
使用 `sudo clashctl init` 或 `sudo clashctl start`。

### 端口被占用
检查是否有其他 Mihomo 实例在运行：`ps aux | grep mihomo`

### 订阅 URL 无法访问
检查网络连接，确认 URL 以 http/https 开头。
