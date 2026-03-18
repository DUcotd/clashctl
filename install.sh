#!/bin/bash
# clashctl + Mihomo 一键安装脚本
# Usage: curl -sL https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh | sudo bash
set -euo pipefail

# ─── Config ───
INSTALL_DIR="/usr/local/bin"
CLASHCTL_REPO="DUcotd/clashctl"
MIHOMO_REPO="MetaCubeX/mihomo"
TIMEOUT=30
MAX_RETRIES=3

# ─── Colors ───
if [ -t 1 ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; RESET='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; CYAN=''; BOLD=''; DIM=''; RESET=''
fi

# ─── Helpers ───
info()  { printf "  ${CYAN}▸${RESET} %s\n" "$*"; }
ok()    { printf "  ${GREEN}✓${RESET} %s\n" "$*"; }
warn()  { printf "  ${YELLOW}⚠${RESET}  %s\n" "$*"; }
err()   { printf "  ${RED}✗${RESET} %s\n" "$*" >&2; }
die()   { err "$@"; exit 1; }

# ─── Args ───
SKIP_MIHOMO=false
SKIP_CLASHCTL=false
CLASHCTL_VERSION="latest"
MIHOMO_VERSION="latest"

usage() {
    cat <<'USAGE'
📦 clashctl 安装脚本

用法:
  install.sh [选项]

选项:
  --clashctl-only       只安装 clashctl
  --mihomo-only         只安装 Mihomo
  --clashctl-version X  指定 clashctl 版本 (默认: latest)
  --mihomo-version X    指定 Mihomo 版本 (默认: latest)
  --install-dir X       安装目录 (默认: /usr/local/bin)
  -h, --help            显示帮助

示例:
  curl -sL https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh | sudo bash
  sudo bash install.sh --clashctl-only
  sudo bash install.sh --clashctl-version v2.3.0
USAGE
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --clashctl-only)    SKIP_MIHOMO=true ;;
        --mihomo-only)      SKIP_CLASHCTL=true ;;
        --clashctl-version) CLASHCTL_VERSION="$2"; shift ;;
        --mihomo-version)   MIHOMO_VERSION="$2"; shift ;;
        --install-dir)      INSTALL_DIR="$2"; shift ;;
        -h|--help)          usage ;;
        *)                  die "未知选项: $1 (用 -h 查看帮助)" ;;
    esac
    shift
done

# ─── Pre-flight ───
[ "$EUID" -eq 0 ] || die "请使用 sudo 运行: sudo bash $0"
command -v curl &>/dev/null || die "需要 curl，请先安装"

OS="$(uname -s)"
ARCH="$(uname -m)"
[ "$OS" = "Linux" ] || die "暂仅支持 Linux，当前: $OS"

case "$ARCH" in
    x86_64|amd64)   GOARCH="amd64" ;;
    aarch64|arm64)  GOARCH="arm64" ;;
    armv7l|armv6l)  GOARCH="armv7" ;;
    *)              die "不支持的架构: $ARCH" ;;
esac

# ─── SHA256 verification ───
verify_checksum() {
    local file="$1" expected_sha="$2"
    if [ -z "$expected_sha" ]; then
        return 0  # no checksum to verify against
    fi
    local actual_sha
    actual_sha="$(sha256sum "$file" | cut -d' ' -f1)"
    if [ "$actual_sha" != "$expected_sha" ]; then
        warn "SHA256 校验失败!"
        warn "  期望: $expected_sha"
        warn "  实际: $actual_sha"
        return 1
    fi
    ok "SHA256 校验通过"
    return 0
}

# Fetch checksums file from release and extract hash for a specific asset
fetch_asset_sha256() {
    local repo="$1" version="$2" asset_name="$3"
    local checksum_url
    if [ "$version" = "latest" ]; then
        checksum_url="https://github.com/$repo/releases/latest/download/checksums-sha256.txt"
    else
        checksum_url="https://github.com/$repo/releases/download/$version/checksums-sha256.txt"
    fi
    local tmpfile
    tmpfile="$(mktemp)"
    if curl -sfL --connect-timeout "$TIMEOUT" "$checksum_url" -o "$tmpfile" 2>/dev/null; then
        # Extract hash for the specific asset (format: "hash  filename")
        local hash
        hash="$(grep "$asset_name" "$tmpfile" | head -1 | awk '{print $1}')"
        rm -f "$tmpfile"
        echo "$hash"
    else
        rm -f "$tmpfile"
        echo ""
    fi
}

# ─── Download to file with retry ───
download_file() {
    local url="$1" output="$2"
    local attempt=0
    while [ "$attempt" -lt "$MAX_RETRIES" ]; do
        if curl -sfL --connect-timeout "$TIMEOUT" --retry 2 "$url" -o "$output"; then
            return 0
        fi
        attempt=$((attempt + 1))
        if [ "$attempt" -lt "$MAX_RETRIES" ]; then
            warn "下载失败，重试 ($attempt/$MAX_RETRIES)..."
            sleep 2
        fi
    done
    return 1
}

# ─── Install clashctl ───
install_clashctl() {
    echo ""
    printf "  ${BOLD}[1/2] 安装 clashctl${RESET}\n"

    # Check local binary first
    if [ -f "./clashctl-linux-amd64" ]; then
        info "检测到本地文件，直接安装"
        mkdir -p "$INSTALL_DIR"
        cp ./clashctl-linux-amd64 "$INSTALL_DIR/clashctl"
        chmod +x "$INSTALL_DIR/clashctl"
        ok "clashctl → $INSTALL_DIR/clashctl"
        return
    fi

    # Resolve version - use temp file to avoid pipefail + set -u issues
    if [ "$CLASHCTL_VERSION" = "latest" ]; then
        info "获取最新版本..."
        local ver_tmp
        ver_tmp="$(mktemp)"
        if curl -sfL --connect-timeout "$TIMEOUT" "https://api.github.com/repos/$CLASHCTL_REPO/releases/latest" -o "$ver_tmp" 2>/dev/null; then
            CLASHCTL_VERSION="$(python3 -c "import json; print(json.load(open('$ver_tmp'))['tag_name'])" 2>/dev/null)" || CLASHCTL_VERSION="latest"
        fi
        rm -f "$ver_tmp"
    fi

    local url
    if [ "$CLASHCTL_VERSION" = "latest" ]; then
        url="https://github.com/$CLASHCTL_REPO/releases/latest/download/clashctl-linux-${GOARCH}"
    else
        url="https://github.com/$CLASHCTL_REPO/releases/download/$CLASHCTL_VERSION/clashctl-linux-${GOARCH}"
    fi

    info "下载 ${DIM}$CLASHCTL_VERSION${RESET}..."
    mkdir -p "$INSTALL_DIR"
    download_file "$url" "$INSTALL_DIR/clashctl" || die "clashctl 下载失败: $url"

    # Verify SHA256 if available
    local expected_sha
    expected_sha="$(fetch_asset_sha256 "$CLASHCTL_REPO" "$CLASHCTL_VERSION" "clashctl-linux-${GOARCH}")"
    if [ -n "$expected_sha" ]; then
        if ! verify_checksum "$INSTALL_DIR/clashctl" "$expected_sha"; then
            rm -f "$INSTALL_DIR/clashctl"
            die "clashctl 下载校验失败，已删除损坏文件"
        fi
    else
        warn "跳过 SHA256 校验（未找到 checksums 文件）"
    fi

    chmod +x "$INSTALL_DIR/clashctl"
    ok "clashctl → $INSTALL_DIR/clashctl"
}

# ─── Install mihomo ───
install_mihomo() {
    echo ""
    printf "  ${BOLD}[2/2] 安装 Mihomo${RESET}\n"

    # Already installed?
    if command -v mihomo &>/dev/null; then
        local ver
        ver="$(mihomo -v 2>/dev/null | head -1)" || ver="unknown"
        if [ "$MIHOMO_VERSION" = "latest" ]; then
            ok "已安装: $ver，跳过"
            return
        else
            info "已安装 $ver，将覆盖为 $MIHOMO_VERSION"
        fi
    fi

    # Resolve release info
    local release_json tmpfile
    tmpfile="$(mktemp)"
    trap 'rm -f "$tmpfile"' RETURN

    if [ "$MIHOMO_VERSION" = "latest" ]; then
        info "获取最新版本..."
        download_file "https://api.github.com/repos/$MIHOMO_REPO/releases/latest" "$tmpfile" \
            || die "无法获取 Mihomo 版本信息"
    else
        info "获取版本 $MIHOMO_VERSION..."
        download_file "https://api.github.com/repos/$MIHOMO_REPO/releases/tags/$MIHOMO_VERSION" "$tmpfile" \
            || die "无法获取 Mihomo $MIHOMO_VERSION"
    fi

    MIHOMO_VERSION="$(python3 -c "import json; print(json.load(open('$tmpfile'))['tag_name'])")"
    info "版本: ${DIM}$MIHOMO_VERSION${RESET}"

    # Find download URL via python3 - filter for linux specifically
    local mihomo_url
    mihomo_url="$(python3 -c "
import json
assets = json.load(open('$tmpfile')).get('assets', [])
arch = '$GOARCH'
skip = ('.deb', '.rpm', '.zst', '.pkg.tar', '.txt', '.sig')
cands = [a for a in assets if 'linux' in a['name'] and arch in a['name'] and not any(s in a['name'] for s in skip)]
if cands:
    gz = [a for a in cands if a['name'].endswith('.gz')]
    print(gz[0]['browser_download_url'] if gz else cands[0]['browser_download_url'])
" 2>/dev/null)" || true

    [ -n "${mihomo_url:-}" ] || die "找不到匹配 $GOARCH 的 Mihomo 二进制"

    mkdir -p "$INSTALL_DIR"
    info "下载 ${DIM}$(basename "$mihomo_url")${RESET}..."

    if [[ "$mihomo_url" == *.gz ]]; then
        local gzfile="$tmpfile.gz"
        download_file "$mihomo_url" "$gzfile" || die "Mihomo 下载失败"
        gzip -d -c "$gzfile" > "$INSTALL_DIR/mihomo" || die "解压失败"
    else
        download_file "$mihomo_url" "$INSTALL_DIR/mihomo" || die "Mihomo 下载失败"
    fi

    chmod +x "$INSTALL_DIR/mihomo"
    ok "mihomo → $INSTALL_DIR/mihomo ($MIHOMO_VERSION)"
}

# ─── Main ───
echo ""
printf "  ${BOLD}📦 clashctl 安装程序${RESET}\n"
printf "  ${DIM}架构: $OS/$GOARCH | 安装目录: $INSTALL_DIR${RESET}\n"

$SKIP_CLASHCTL || install_clashctl
$SKIP_MIHOMO   || install_mihomo

# ─── Done ───
echo ""
printf "  ${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
printf "  ${GREEN}✅ 安装完成！${RESET}\n"
echo ""
printf "  ${BOLD}开始使用：${RESET}\n"
printf "    ${CYAN}sudo clashctl init${RESET}    # 交互式配置向导\n"
printf "    ${CYAN}sudo clashctl doctor${RESET}  # 环境自检\n"
echo ""

# Show version
"$INSTALL_DIR/clashctl" version 2>/dev/null || true
