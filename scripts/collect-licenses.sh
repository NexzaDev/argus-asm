#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: scripts/collect-licenses.sh OUTDIR [PACKAGE]" >&2
  exit 2
}

[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage
OUT="$1"
PKG="${2:-./cmd/argus-asm}"

rm -rf "$OUT"
mkdir -p "$OUT"
index="$OUT/INDEX.txt"
: > "$index"

go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}} {{.Version}}{{end}}{{end}}' "$PKG" | sort -u | while read -r path version; do
  [ -n "$path" ] || continue
  dir="$(go list -m -f '{{.Dir}}' "$path")"
  if [ -z "$dir" ] || [ ! -d "$dir" ]; then
    echo "error: no source directory for $path $version" >&2
    exit 1
  fi
  safe="$(printf '%s' "$path" | tr '/' '_')"
  files="$(cd "$dir" && find . -maxdepth 1 -type f \( -iname 'LICENSE*' -o -iname 'COPYING*' -o -iname 'NOTICE*' \) | sed 's#^\./##' | sort)"
  if [ -z "$files" ]; then
    echo "error: no LICENSE, COPYING, or NOTICE file found for $path $version in $dir" >&2
    exit 1
  fi
  mkdir -p "$OUT/$safe"
  while IFS= read -r f; do
    cp "$dir/$f" "$OUT/$safe/$f"
  done <<< "$files"
  printf '%s %s: %s\n' "$path" "$version" "$(echo "$files" | tr '\n' ' ')" >> "$index"
done

count="$(wc -l < "$index" | tr -d ' ')"
if [ "$count" -eq 0 ]; then
  echo "error: no third-party modules found for $PKG" >&2
  exit 1
fi
echo "collected license files for $count module(s) into $OUT"
