#!/bin/sh
set -eu
prefix="${HOME}/.local"
if [ "${1:-}" = "--prefix" ]; then
  [ "$#" -ge 3 ] || { echo 'usage: install.sh [--prefix DIR] BINARY' >&2; exit 2; }
  prefix=$2
  shift 2
fi
[ "$#" -eq 1 ] || { echo 'usage: install.sh [--prefix DIR] BINARY' >&2; exit 2; }
binary=$1
[ -f "$binary" ] && [ -x "$binary" ] || { echo 'binary must be an executable file' >&2; exit 2; }
"$binary" --version
mkdir -p "$prefix/bin"
stage=$(mktemp "$prefix/bin/.mtask.XXXXXX")
trap 'test ! -f "$stage" || rm "$stage"' EXIT HUP INT TERM
cp "$binary" "$stage"
chmod 0755 "$stage"
mv -f "$stage" "$prefix/bin/mtask"
printf 'Installed %s\n' "$prefix/bin/mtask"
