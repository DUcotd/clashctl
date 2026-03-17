# clashctl 开发文档

## 项目结构

```
clashctl/
├── main.go                 # 入口
├── cmd/                    # Cobra 命令定义
│   ├── root.go
│   ├── init.go             # TUI 向导
│   ├── export.go           # 配置导出
│   ├── start.go / stop.go / restart.go
│   ├── status.go
│   ├── doctor.go
│   ├── nodes.go
│   └── config.go
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
│   ├── system/             # 系统工具
│   │   ├── fs.go           # 文件系统操作
│   │   ├── exec.go         # 命令执行
│   │   ├── privilege.go    # 权限检查
│   │   └── network.go      # 网络工具
│   └── ui/                 # Bubble Tea TUI
│       ├── wizard.go       # 主模型
│       ├── screens.go      # 页面渲染
│       ├── executor.go     # 执行管线
│       ├── styles.go       # lipgloss 样式
│       └── state.go        # 状态定义
├── assets/
│   └── clashctl.service.tpl # systemd 服务模板
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
