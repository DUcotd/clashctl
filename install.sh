#!/bin/bash
# myproxy 一键安装脚本
set -e

BINARY_URL="请替换为你的下载链接"  # 例如: https://github.com/you/myproxy/releases/latest/download/myproxy-linux-amd64
INSTALL_PATH="/usr/local/bin/myproxy"

echo "📦 安装 myproxy..."

# Check root
if [ "$EUID" -ne 0 ]; then
    echo "❌ 请使用 sudo 运行此脚本"
    exit 1
fi

# Check arch
ARCH=$(uname -m)
if [ "$ARCH" != "x86_64" ]; then
    echo "❌ 暂仅支持 x86_64 架构，当前: $ARCH"
    exit 1
fi

# Check OS
OS=$(uname -s)
if [ "$OS" != "Linux" ]; then
    echo "❌ 暂仅支持 Linux，当前: $OS"
    exit 1
fi

# If binary is local (script used with bundled binary)
if [ -f "./myproxy-linux-amd64" ]; then
    echo "  从本地文件安装..."
    cp ./myproxy-linux-amd64 "$INSTALL_PATH"
else
    echo "  下载中..."
    curl -sL "$BINARY_URL" -o "$INSTALL_PATH"
fi

chmod +x "$INSTALL_PATH"

# Verify
if command -v myproxy &>/dev/null; then
    echo "✅ 安装成功！"
    echo ""
    myproxy --help
    echo ""
    echo "开始使用："
    echo "  sudo myproxy init    # 交互式配置向导"
    echo "  sudo myproxy doctor  # 环境自检"
else
    echo "⚠️  安装完成但未在 PATH 中找到，请检查 $INSTALL_PATH"
fi
