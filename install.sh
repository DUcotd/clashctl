#!/bin/bash
# clashctl + Mihomo 一键安装脚本
#
# 安全安装方式（推荐）：
#   curl -sLO https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh
#   # 可选：校验脚本哈希
#   chmod +x install.sh
#   sudo ./install.sh
#
# 管道安装方式（便捷但有风险）：
#   curl -sL https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh | sudo bash
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

# Check for systemd (required for service management)
check_systemd() {
    if [ ! -d "/run/systemd/system" ]; then
        warn "未检测到 systemd，将跳过服务集成"
        warn "  仅安装二进制文件，需手动管理服务"
        return 1
    fi
    return 0
}

HAS_SYSTEMD=true
check_systemd || HAS_SYSTEMD=false

OS="$(uname -s)"
ARCH="$(uname -m)"
[ "$OS" = "Linux" ] || die "暂仅支持 Linux，当前: $OS"

case "$ARCH" in
    x86_64|amd64)   GOARCH="amd64" ;;
    aarch64|arm64)  GOARCH="arm64" ;;
    armv7l|armv6l)  GOARCH="arm" ;;
    *)              die "不支持的架构: $ARCH" ;;
esac

# ─── URL validation ───
validate_url() {
    local url="$1"
    # Only allow HTTPS URLs from known GitHub domains
    if [[ "$url" =~ ^https://(github\.com|raw\.githubusercontent\.com|api\.github\.com)/ ]]; then
        return 0
    fi
    warn "URL 来源非官方 GitHub 域名: $url"
    return 1
}

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
# Returns the hash if found, or empty string if checksums file doesn't exist (404)
# On network error, outputs "ERROR" to signal failure
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
    local http_code
    http_code=$(curl -sL --connect-timeout "$TIMEOUT" -w "%{http_code}" -o "$tmpfile" "$checksum_url" 2>/dev/null)
    if [ "$http_code" = "200" ]; then
        # Extract hash for the specific asset (format: "hash  filename")
        local hash
        hash="$(grep "$asset_name" "$tmpfile" | head -1 | awk '{print $1}')"
        rm -f "$tmpfile"
        echo "$hash"
    elif [ "$http_code" = "404" ]; then
        # Checksums file doesn't exist (old release), return empty
        rm -f "$tmpfile"
        echo ""
    else
        # Network error or other failure
        rm -f "$tmpfile"
        echo "ERROR"
    fi
}

# ─── Rollback tracking ───
ROLLBACK_FILES=()
ROLLBACK_ACTIONS=()

rollback_add_file() {
    ROLLBACK_FILES+=("$1")
}

rollback_add_action() {
    ROLLBACK_ACTIONS+=("$1")
}

do_rollback() {
    if [ ${#ROLLBACK_FILES[@]} -eq 0 ] && [ ${#ROLLBACK_ACTIONS[@]} -eq 0 ]; then
        return
    fi
    warn "执行回滚操作..."
    # Remove installed files
    for f in "${ROLLBACK_FILES[@]}"; do
        if [ -f "$f" ]; then
            rm -f "$f"
            info "已删除: $f"
        fi
    done
    # Execute rollback actions
    for action in "${ROLLBACK_ACTIONS[@]}"; do
        eval "$action" 2>/dev/null || true
    done
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

    local local_asset="./clashctl-linux-${GOARCH}"

    # Check local binary first, but require explicit opt-in to trust local files.
    if [ -f "$local_asset" ]; then
        if [ "${CLASHCTL_INSTALL_ALLOW_LOCAL_BINARY:-}" != "1" ]; then
            die "检测到本地文件 $local_asset，但默认拒绝信任本地二进制。若确认可信，请显式设置 CLASHCTL_INSTALL_ALLOW_LOCAL_BINARY=1 后重试"
        fi
        warn "已显式允许使用本地二进制，请自行确认其来源可信"
        mkdir -p "$INSTALL_DIR"
        cp "$local_asset" "$INSTALL_DIR/clashctl"
        chmod +x "$INSTALL_DIR/clashctl"
        rollback_add_file "$INSTALL_DIR/clashctl"
        ok "clashctl → $INSTALL_DIR/clashctl"
        return
    fi

    # Resolve version - use temp file to avoid pipefail + set -u issues
    if [ "$CLASHCTL_VERSION" = "latest" ]; then
        info "获取最新版本..."
        local ver_tmp
        ver_tmp="$(mktemp)"
        if curl -sfL --connect-timeout "$TIMEOUT" "https://api.github.com/repos/$CLASHCTL_REPO/releases/latest" -o "$ver_tmp" 2>/dev/null; then
            CLASHCTL_VERSION="$(grep -m1 '"tag_name"' "$ver_tmp" | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')" || CLASHCTL_VERSION="latest"
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
    
    # Create temp file for download, then move after verification
    local tmpfile
    tmpfile="$(mktemp)"
    download_file "$url" "$tmpfile" || { rm -f "$tmpfile"; die "clashctl 下载失败: $url"; }

    # Verify SHA256 if available
    local expected_sha
    expected_sha="$(fetch_asset_sha256 "$CLASHCTL_REPO" "$CLASHCTL_VERSION" "clashctl-linux-${GOARCH}")"
    if [ "$expected_sha" = "ERROR" ]; then
        rm -f "$tmpfile"
        die "下载校验和文件失败，无法验证二进制完整性"
    elif [ -n "$expected_sha" ]; then
        if ! verify_checksum "$tmpfile" "$expected_sha"; then
            rm -f "$tmpfile"
            die "clashctl 下载校验失败，已删除损坏文件"
        fi
    else
        warn "跳过 SHA256 校验（未找到 checksums 文件）"
    fi

    chmod +x "$tmpfile"
    mv "$tmpfile" "$INSTALL_DIR/clashctl"
    rollback_add_file "$INSTALL_DIR/clashctl"
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

    MIHOMO_VERSION="$(grep -m1 '"tag_name"' "$tmpfile" | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
    info "版本: ${DIM}$MIHOMO_VERSION${RESET}"

    # Find download URL — grep for linux + arch in asset names, prefer .gz
    local mihomo_url
    # First try .gz
    mihomo_url="$(grep -o '"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]*"' "$tmpfile" \
        | sed 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' \
        | grep -i "linux" | grep -i "$GOARCH" | grep '\.gz$' | grep -v 'pkg\.tar' | head -1)" || true
    # Fallback: any matching binary (not .deb/.rpm/.zst/.txt/.sig)
    if [ -z "$mihomo_url" ]; then
        mihomo_url="$(grep -o '"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]*"' "$tmpfile" \
            | sed 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' \
            | grep -i "linux" | grep -i "$GOARCH" \
            | grep -v '\.deb$' | grep -v '\.rpm$' | grep -v '\.zst$' | grep -v '\.txt$' | grep -v '\.sig$' | head -1)" || true
    fi

    [ -n "${mihomo_url:-}" ] || die "找不到匹配 $GOARCH 的 Mihomo 二进制"

    local asset_name expected_sha
    asset_name="$(basename "$mihomo_url")"
    expected_sha="$(fetch_asset_sha256 "$MIHOMO_REPO" "$MIHOMO_VERSION" "$asset_name")"
    if [ "$expected_sha" = "ERROR" ]; then
        die "下载 Mihomo 校验和文件失败，无法验证二进制完整性"
    fi
    if [ -z "$expected_sha" ]; then
        warn "发布缺少 $asset_name 的 SHA256 校验值"
        warn "  将跳过校验继续安装（安全性降低）"
    fi

    mkdir -p "$INSTALL_DIR"
    info "下载 ${DIM}$(basename "$mihomo_url")${RESET}..."

    # Download to temp file first for verification
    local mihomo_tmpfile
    mihomo_tmpfile="$(mktemp)"
    
    if [[ "$mihomo_url" == *.gz ]]; then
        local gzfile="$mihomo_tmpfile.gz"
        download_file "$mihomo_url" "$gzfile" || { rm -f "$mihomo_tmpfile" "$gzfile"; die "Mihomo 下载失败"; }
        verify_checksum "$gzfile" "$expected_sha" || { rm -f "$mihomo_tmpfile" "$gzfile"; die "Mihomo 下载校验失败，已删除损坏文件"; }
        gzip -d -c "$gzfile" > "$mihomo_tmpfile" || { rm -f "$mihomo_tmpfile" "$gzfile"; die "解压失败"; }
        rm -f "$gzfile"
    else
        download_file "$mihomo_url" "$mihomo_tmpfile" || { rm -f "$mihomo_tmpfile"; die "Mihomo 下载失败"; }
        verify_checksum "$mihomo_tmpfile" "$expected_sha" || { rm -f "$mihomo_tmpfile"; die "Mihomo 下载校验失败，已删除损坏文件"; }
    fi

    # Verify binary works before installing
    chmod +x "$mihomo_tmpfile"
    if ! "$mihomo_tmpfile" -v >/dev/null 2>&1; then
        rm -f "$mihomo_tmpfile"
        die "下载的 Mihomo 二进制不可用"
    fi

    # Move to final location
    mv "$mihomo_tmpfile" "$INSTALL_DIR/mihomo"
    rollback_add_file "$INSTALL_DIR/mihomo"
    ok "mihomo → $INSTALL_DIR/mihomo ($MIHOMO_VERSION)"
}

# ─── Main ───
echo ""
printf "  ${BOLD}📦 clashctl 安装程序${RESET}\n"
printf "  ${DIM}架构: $OS/$GOARCH | 安装目录: $INSTALL_DIR${RESET}\n"

# Install with error handling and rollback
if ! $SKIP_CLASHCTL; then
    if ! install_clashctl; then
        do_rollback
        die "clashctl 安装失败"
    fi
fi

if ! $SKIP_MIHOMO; then
    if ! install_mihomo; then
        do_rollback
        die "Mihomo 安装失败"
    fi
fi

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
