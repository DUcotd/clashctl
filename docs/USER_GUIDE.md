# clashctl 用户指南

## 简介

clashctl 是一个运行在 Linux 终端的交互式工具，帮助你通过输入机场订阅 URL 快速完成 Mihomo 代理的配置、启动与管理。

## 前提条件

- **Mihomo** 已安装并在 PATH 中
  - 下载：https://github.com/MetaCubeX/mihomo/releases
- **root 权限**（TUN 模式必需）
- **systemd**（推荐，用于服务管理）

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
查看运行状态、代理组、当前节点。

### clashctl doctor
环境自检（9项检查）。

```bash
sudo clashctl doctor          # 包含 TUN 检查
clashctl doctor --tun=false   # 跳过 TUN 检查
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
| Mihomo 配置 | `/etc/mihomo/config.yaml` |
| Provider 缓存 | `/etc/mihomo/providers/airport.yaml` |
| systemd 服务 | `/etc/systemd/system/clashctl-mihomo.service` |

## 常见问题

### 未找到 mihomo
请从 GitHub Releases 下载并确保在 PATH 中。

### TUN 模式需要 root
使用 `sudo clashctl init` 或 `sudo clashctl start`。

### 端口被占用
检查是否有其他 Mihomo 实例在运行：`ps aux | grep mihomo`

### 订阅 URL 无法访问
检查网络连接，确认 URL 以 http/https 开头。
