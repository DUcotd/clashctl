# clashctl 开发文档

## 项目结构

```
clashctl/
├── main.go                 # 入口
├── cmd/                    # Cobra 命令定义
│   ├── root.go
│   ├── init.go             # TUI 向导
│   ├── service.go          # 安装/启动/停止/重启/状态
│   ├── config.go           # 导入/导出/查看配置
│   ├── nodes.go            # 节点管理
│   ├── doctor.go           # 环境诊断
│   ├── backup.go           # 备份/恢复命令
│   ├── compat.go           # 兼容旧命令入口
│   ├── update.go           # 自更新逻辑（含 self 兼容别名）
│   ├── output*.go          # 统一文本/JSON 输出辅助
│   └── legacy.go           # 迁移提示辅助
├── internal/
│   ├── app/                # 应用初始化
│   │   └── bootstrap.go
│   ├── core/               # 核心模型 & 逻辑
│   │   ├── model.go        # AppConfig
│   │   ├── mihomo_config.go # Mihomo YAML 结构体
│   │   ├── builder.go      # 配置构建
│   │   ├── renderer.go     # YAML 序列化
│   │   ├── validator.go    # 参数校验
│   │   └── defaults.go     # 默认值常量
│   ├── config/             # 配置文件读写
│   │   ├── loader.go       # 读取 & 备份
│   │   └── writer.go       # 写入 & 校验
│   ├── mihomo/             # Mihomo 交互
│   │   ├── api.go          # Controller REST API
│   │   ├── api_enhanced.go # 增强 API（多组、延迟）
│   │   ├── process.go      # 进程管理
│   │   ├── service.go      # systemd 服务
│   │   └── doctor.go       # 环境诊断
│   ├── nodes/              # nodes CLI / TUI 共用操作
│   ├── releases/           # GitHub Release 获取与下载回退
│   ├── setup/              # init / import 共用落地流程
│   ├── system/             # 系统工具
│   │   ├── fs.go           # 文件系统操作
│   │   ├── exec.go         # 命令执行
│   │   ├── privilege.go    # 权限检查
│   │   └── network.go      # 网络工具
│   └── ui/                 # Bubble Tea TUI
│       ├── wizard.go       # init 向导模型
│       ├── screens.go      # 向导页面渲染
│       ├── node_manager.go # 节点管理模型
│       ├── node_screens.go # 节点管理页面渲染
│       ├── setup_service.go # 配置/启动流式执行服务
│       ├── node_service.go # 节点管理与测速服务适配层
│       ├── styles.go       # lipgloss 样式
│       └── state.go        # 状态定义
├── assets/
│   └── clashctl.service.tpl # 文档参考模板
└── docs/
    ├── USER_GUIDE.md
    └── DEVELOPMENT.md
```

## 技术栈

- Go 1.21+
- Cobra (CLI 框架)
- Bubble Tea (TUI)
- lipgloss (样式)
- yaml.v3 (YAML 序列化)

## 构建

```bash
go build -o clashctl .
```

## 测试

```bash
go test ./...
```

## 开发阶段

| 阶段 | 内容 | 状态 |
|------|------|------|
| 1 | 脚手架、模型、export | ✅ |
| 2 | Bubble Tea TUI 向导 | ✅ |
| 3 | 系统集成（mihomo、systemd） | ✅ |
| 4 | API 管理增强 | ✅ |
| 5 | doctor 完善 & 测试 | ✅ |

## 代码规范

- 所有公开函数和结构体必须有注释
- 错误信息必须是"问题 + 原因 + 建议"三段式
- 不允许把所有逻辑写进 main.go
- 不允许用字符串硬拼 YAML
- 不允许省略错误处理
