#!/bin/sh
# Download and verify an official release, then replace the binary atomically.
set -eu

repo=https://github.com/raywcm/mtask
prefix=${HOME:?HOME is not set}/.local
version=latest

usage() {
  cat <<'EOF'
Usage: install.sh [--version VERSION] [--prefix DIR]

Install the latest mtask release to ~/.local/bin.
  --version VERSION  Install a specific release, for example v0.4.4
  --prefix DIR       Install to DIR/bin without changing shell configuration
  -h, --help         Show this help
EOF
}

fail() { printf 'mtask: %s\n' "$*" >&2; exit 1; }

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version|--prefix)
      [ "$#" -ge 2 ] && [ -n "$2" ] || fail "$1 needs a value"
      case "$1" in --version) version=$2 ;; --prefix) prefix=$2 ;; esac
      shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; fail "unknown argument: $1" ;;
  esac
done

case "$(uname -s)" in
  Linux) platform=linux ;;
  Darwin) platform=darwin ;;
  *) fail 'supported systems: Linux and macOS' ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) fail 'supported architectures: amd64 and arm64' ;;
esac

command -v curl >/dev/null 2>&1 || fail 'curl is required'
command -v tar >/dev/null 2>&1 || fail 'tar is required'
if command -v sha256sum >/dev/null 2>&1; then
  hash=sha256sum
elif command -v shasum >/dev/null 2>&1; then
  hash=shasum
else
  fail 'sha256sum or shasum is required'
fi

download() {
  curl --fail --silent --show-error --location --retry 3 \
    --connect-timeout 15 --max-time 180 --proto '=https' --proto-redir '=https' "$@"
}
if [ "$version" = latest ]; then
  url=$(download --output /dev/null --write-out '%{url_effective}' "$repo/releases/latest") || fail 'cannot find the latest release'
  case "$url" in
    "$repo/releases/tag/"*) version=${url##*/} ;;
    *) fail 'unexpected latest-release URL' ;;
  esac
fi
case "$version" in v*) ;; *) version=v$version ;; esac
# Keep the tag safe for URLs and paths; only version-shaped tags are accepted.
case "$version" in
  *[!a-zA-Z0-9.+-]*|v|v[!0-9]*) fail 'invalid release version' ;;
esac
case "$version" in v[0-9]*.[0-9]*.[0-9]*) ;; *) fail 'use a version such as v0.4.4' ;; esac

temp=$(mktemp -d "${TMPDIR:-/tmp}/mtask-install.XXXXXX")
stage=
cleanup() {
  [ -z "$stage" ] || rm -f "$stage"
  rm -rf "$temp"
}
trap cleanup 0
trap 'exit 1' HUP INT TERM

archive=mtask_${version#v}_${platform}_${arch}.tar.gz
printf 'Downloading mtask %s for %s/%s...\n' "$version" "$platform" "$arch"
download --output "$temp/$archive" "$repo/releases/download/$version/$archive" || fail 'archive download failed'
download --output "$temp/checksums.txt" "$repo/releases/download/$version/checksums.txt" || fail 'checksum download failed'
expected=$(awk -v name="$archive" '$2 == name { print $1; count++ } END { if (count != 1) exit 1 }' "$temp/checksums.txt") || fail 'archive checksum is missing or duplicated'
[ "${#expected}" -eq 64 ] || fail 'invalid SHA-256 checksum'
case "$expected" in *[!0-9a-fA-F]*) fail 'invalid SHA-256 checksum' ;; esac
if [ "$hash" = sha256sum ]; then
  actual=$(sha256sum "$temp/$archive")
else
  actual=$(shasum -a 256 "$temp/$archive")
fi
actual=${actual%% *}
[ "$actual" = "$expected" ] || fail 'SHA-256 verification failed; existing installation was not changed'
tar -xzf "$temp/$archive" -C "$temp" mtask || fail 'cannot extract mtask'
[ -f "$temp/mtask" ] && [ ! -L "$temp/mtask" ] || fail 'invalid binary in archive'
chmod 0755 "$temp/mtask"
"$temp/mtask" --version || fail 'the downloaded binary cannot run on this system'

case "$prefix" in /*) ;; *) prefix=$PWD/$prefix ;; esac
mkdir -p "$prefix/bin"
stage=$(mktemp "$prefix/bin/.mtask.XXXXXX")
cp "$temp/mtask" "$stage"
chmod 0755 "$stage"
mv -f "$stage" "$prefix/bin/mtask"
stage=
printf 'Installed %s/bin/mtask\n' "$prefix"
case ":${PATH:-}:" in
  *":$prefix/bin:"*) ;;
  *) printf 'Add %s/bin to PATH, then run mtask.\n' "$prefix" ;;
esac
