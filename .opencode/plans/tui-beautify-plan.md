# TUI 美化计划

## 项目现状

- **技术栈**: Bubble Tea v0.25.0 + Bubbles v0.18.0 + Lipgloss v0.9.1
- **两大界面**: 配置向导 (Wizard) + 节点管理 (Node Manager)
- **样式文件**: `internal/ui/styles.go` 定义了 20+ 种 lipgloss 样式

## 当前问题

1. **色彩方案** - 颜色较杂乱，缺乏统一主题色
2. **状态栏** - 纯 ASCII 边框 (`┌─┐`)，缺乏视觉层次
3. **帮助页面** - 纯文本堆砌，无分组视觉区分
4. **列表项** - 节点列表信息密度低，缺少结构化展示
5. **进度展示** - 测速进度条简陋，无动画过渡
6. **卡片边框** - 所有卡片使用同一紫色圆角边框，单调
7. **标题栏** - 缺少分隔线和装饰元素
8. **空状态** - 无数据时提示过于简单

## 美化方案

### 1. 统一色彩主题

**目标**: 建立一致的暗色主题色彩体系

**现有问题**:
- 颜色硬编码散落在各处（`#FF6B9D`, `#7C3AED`, `#E2E8F0` 等）
- 没有语义化的颜色命名

**改进**:
- 引入常量定义：`colorBgBase`, `colorPrimary`, `colorSecondary`, `colorAccent`, `colorSuccess`, `colorWarning`, `colorError`
- 新增样式：
  - `TitleDividerStyle` - 标题分隔线
  - `HeaderBarStyle` - 带背景的标题栏
  - `InputFocusedStyle` - 输入框聚焦高亮
  - `SelectedBarStyle` - 选中项左侧指示条
  - `KeyStyle` / `KeySepStyle` - 快捷键按键样式
  - `ProgressBarFullStyle` / `ProgressBarEmptyStyle` - 进度条
  - `StatusBarStyle` / `StatusHelpStyle` / `StatusDividerStyle` - 状态栏
  - `DetailLabelStyle` / `DetailValueStyle` - 详情视图标签
  - `CardHeaderStyle` - 卡片头部
  - `ConfirmStyle` - 确认对话框
  - `SelectorActiveStyle` / `SelectorInactiveStyle` - 来源选择器
  - `HelpSectionStyle` - 帮助分区标题
  - `EmptyStateIconStyle` / `EmptyStateTextStyle` - 空状态
  - `BoxSuccessStyle` / `BoxWarningStyle` / `BoxErrorStyle` - 不同语义的卡片边框

### 2. 增强列表渲染

**目标**: 节点列表改为结构化布局

**现有问题**:
- 节点行只是简单拼接字符串，没有对齐
- 选中项只有 `▸` 前缀，不够醒目

**改进**:
- 节点列表使用 `SelectedBarStyle` 为选中项添加左侧指示条
- 协议标签、延迟值右对齐
- 空状态使用图标 + 多行说明

### 3. 改进状态栏

**目标**: 替换 ASCII 边框为 lipgloss 样式

**现有问题**:
- `renderStatusBar()` 使用 `┌─┐` ASCII 字符构建
- 状态信息和快捷键提示没有视觉区分

**改进**:
- 使用 `StatusBarStyle` + `StatusHelpStyle` 构建双色状态栏
- 使用 `StatusDividerStyle` 添加彩色分隔符

### 4. 优化帮助页面

**目标**: 分组标题 + 键盘样式快捷键

**现有问题**:
- 纯文本，所有快捷键没有视觉区分
- 分组之间只有空行

**改进**:
- 分组标题使用 `HelpSectionStyle` 彩色背景
- 快捷键使用 `KeyStyle` 渲染为类似 `<kbd>` 的按钮效果
- 分隔符使用 `KeySepStyle`

### 5. 美化进度展示

**目标**: 彩色渐变进度条

**现有问题**:
- 进度条只有 `█` 和 `░` 字符
- 无百分比高亮

**改进**:
- 使用 `ProgressBarFullStyle` 渲染已填充部分
- 使用 `ProgressTextStyle` 高亮百分比数字

### 6. 改进空状态和错误提示

**目标**: 图标 + 多行说明 + 操作建议

**现有问题**:
- 空状态只有一行 `WarningStyle.Render("未找到任何代理组")`

**改进**:
- 使用 `EmptyStateIconStyle` 渲染大图标
- 使用 `EmptyStateTextStyle` 渲染说明文字
- 使用 `BoxWarningStyle` / `BoxErrorStyle` 区分语义

### 7. 标题栏增强

**目标**: 分隔线 + 副标题 + 步骤导航点

**现有问题**:
- 步骤指示器只是纯文本 `步骤 1/7: 选择订阅来源`

**改进**:
- 步骤指示器使用 `StepDotActiveStyle` / `StepDotDoneStyle` / `StepDotInactiveStyle` 渲染进度点
- 标题使用 `CardHeaderStyle` 增强

### 8. 细节打磨

- 输入框聚焦时切换为 `InputFocusedStyle`
- 确认对话框使用 `ConfirmStyle` + `BoxWarningStyle`
- 节点详情使用 `DetailLabelStyle` / `DetailValueStyle` 双列对齐
- 来源选择器使用 `SelectorActiveStyle` / `SelectorInactiveStyle`

## 文件修改清单

| 文件 | 修改内容 |
|------|---------|
| `internal/ui/styles.go` | 重构色彩体系，新增 20+ 样式，引入颜色常量 |
| `internal/ui/layout.go` | 增强 `renderCard`，添加分隔线和语义化边框 |
| `internal/ui/state.go` | 步骤指示器改用进度点样式 |
| `internal/ui/screens.go` | 向导各页面使用新样式 |
| `internal/ui/node_screens.go` | 列表、状态栏、帮助页、空状态美化 |
| `internal/ui/node_manager.go` | View 逻辑微调，适配新样式 |

## 风险与注意事项

1. **向后兼容**: 所有现有样式变量保留，只新增不删除
2. **测试覆盖**: 运行 `go test ./internal/ui/...` 确保无回归
3. **终端兼容**: 新样式仅使用 lipgloss 标准特性，兼容主流终端
4. **宽度适配**: 新增的装饰元素需考虑窄终端场景
