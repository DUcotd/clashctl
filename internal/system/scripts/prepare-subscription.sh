#!/bin/sh
set -eu

if [ "$#" -lt 2 ]; then
  printf '%s\n' "usage: prepare-subscription.sh <url> <output-dir> [timeout] [user-agent]" >&2
  exit 2
fi

URL="$1"
OUT_DIR="$2"
TIMEOUT="${3:-15}"
USER_AGENT="${4:-clashctl/dev}"

# Validate URL format (only allow http/https)
case "$URL" in
  http://*|https://*)
    # OK
    ;;
  *)
    printf '%s\n' "error: only http/https URLs are allowed" >&2
    exit 1
    ;;
esac

# Reject URLs with shell metacharacters
case "$URL" in
  *";"*|"|"*|"`"*|'$('*|'&&"*|'||'*)
    printf '%s\n' "error: URL contains invalid characters" >&2
    exit 1
    ;;
esac

CONTENT_FILE="$OUT_DIR/subscription.txt"
INFO_FILE="$OUT_DIR/subscription.info"

mkdir -p "$OUT_DIR"

unset http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY no_proxy NO_PROXY

FETCHER=""
if command -v curl >/dev/null 2>&1; then
  FETCHER="curl"
  # Use -- to prevent URL from being interpreted as options
  curl --noproxy '*' --connect-timeout "$TIMEOUT" --max-time "$TIMEOUT" -fsSL -A "$USER_AGENT" -- "$URL" -o "$CONTENT_FILE"
elif command -v wget >/dev/null 2>&1; then
  FETCHER="wget"
  wget --no-proxy -T "$TIMEOUT" -U "$USER_AGENT" -qO "$CONTENT_FILE" -- "$URL"
else
  printf '%s\n' "curl and wget are not available" >&2
  exit 1
fi

BYTES=$(wc -c < "$CONTENT_FILE" | tr -d ' ')

{
  printf 'url=%s\n' "$URL"
  printf 'fetcher=%s\n' "$FETCHER"
  printf 'bytes=%s\n' "$BYTES"
} > "$INFO_FILE"

printf '%s\n' "$CONTENT_FILE"
