#!/bin/bash
# clashctl 一键安装脚本（包含 mihomo 内核）
set -e

CLASHCTL_URL="https://github.com/DUcotd/clashctl/releases/latest/download/clashctl-linux-amd64"
MIHOMO_API="https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"
INSTALL_DIR="/usr/local/bin"

echo "📦 安装 clashctl + Mihomo..."

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

# --- Install clashctl ---
echo ""
echo "  [1/2] 安装 clashctl..."
if [ -f "./clashctl-linux-amd64" ]; then
    echo "    从本地文件安装..."
    cp ./clashctl-linux-amd64 "$INSTALL_DIR/clashctl"
else
    echo "    下载中..."
    curl -sL "$CLASHCTL_URL" -o "$INSTALL_DIR/clashctl"
fi
chmod +x "$INSTALL_DIR/clashctl"
echo "    ✅ clashctl → $INSTALL_DIR/clashctl"

# --- Install mihomo ---
echo ""
echo "  [2/2] 安装 Mihomo 内核..."

# Check if already installed
if command -v mihomo &>/dev/null; then
    MIHOMO_VER=$(mihomo -v 2>/dev/null | head -1 || echo "unknown")
    echo "    已安装: $MIHOMO_VER，跳过"
else
    echo "    获取最新版本..."
    RELEASE_JSON=$(curl -sL "$MIHOMO_API")
    MIHOMO_TAG=$(echo "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//' | sed 's/".*//')
    echo "    版本: $MIHOMO_TAG"

    # Try .gz first (most common for mihomo releases)
    MIHOMO_URL=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url": *"[^"]*linux-amd64[^"]*\.gz"' | grep -v 'pkg.tar' | head -1 | sed 's/.*"browser_download_url": *"//' | sed 's/".*//')

    if [ -n "$MIHOMO_URL" ]; then
        echo "    下载中 (gz)..."
        curl -sL "$MIHOMO_URL" | gzip -d > "$INSTALL_DIR/mihomo"
    else
        # Try uncompressed
        MIHOMO_URL=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url": *"[^"]*linux-amd64[^"]*"' | grep -v '.gz\|.zip\|.deb\|.rpm\|.zst' | head -1 | sed 's/.*"browser_download_url": *"//' | sed 's/".*//')
        if [ -n "$MIHOMO_URL" ]; then
            echo "    下载中..."
            curl -sL "$MIHOMO_URL" -o "$INSTALL_DIR/mihomo"
        fi
    fi

    if [ -f "$INSTALL_DIR/mihomo" ]; then
        chmod +x "$INSTALL_DIR/mihomo"
        echo "    ✅ mihomo → $INSTALL_DIR/mihomo"
    else
        echo "    ⚠️  Mihomo 下载失败，请手动安装"
        echo "    下载地址: https://github.com/MetaCubeX/mihomo/releases/latest"
    fi
fi

# --- Summary ---
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 安装完成！"
echo ""
echo "开始使用："
echo "  sudo clashctl init    # 交互式配置向导"
echo "  sudo clashctl doctor  # 环境自检"
echo ""
clashctl version 2>/dev/null || true
