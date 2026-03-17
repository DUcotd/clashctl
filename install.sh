#!/bin/bash
# clashctl 一键安装脚本
set -e

BINARY_URL="https://github.com/DUcotd/clashctl/releases/latest/download/clashctl-linux-amd64"
INSTALL_PATH="/usr/local/bin/clashctl"

echo "📦 安装 clashctl..."

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
if [ -f "./clashctl-linux-amd64" ]; then
    echo "  从本地文件安装..."
    cp ./clashctl-linux-amd64 "$INSTALL_PATH"
else
    echo "  下载中..."
    curl -sL "$BINARY_URL" -o "$INSTALL_PATH"
fi

chmod +x "$INSTALL_PATH"

# Verify
if command -v clashctl &>/dev/null; then
    echo "✅ 安装成功！"
    echo ""
    clashctl --help
    echo ""
    echo "开始使用："
    echo "  sudo clashctl init    # 交互式配置向导"
    echo "  sudo clashctl doctor  # 环境自检"
else
    echo "⚠️  安装完成但未在 PATH 中找到，请检查 $INSTALL_PATH"
fi
