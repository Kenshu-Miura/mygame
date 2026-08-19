#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist_dir="${project_root}/dist"

rm -rf "${dist_dir}"
mkdir -p "${dist_dir}"

cd "${project_root}"
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "${dist_dir}/mygame.wasm" .

go_root="$(go env GOROOT)"
wasm_exec="${go_root}/lib/wasm/wasm_exec.js"
if [[ ! -f "${wasm_exec}" ]]; then
  wasm_exec="${go_root}/misc/wasm/wasm_exec.js"
fi
if [[ ! -f "${wasm_exec}" ]]; then
  echo "wasm_exec.js was not found under ${go_root}" >&2
  exit 1
fi

cp "${wasm_exec}" "${dist_dir}/wasm_exec.js"
cp "${project_root}/web"/*.html "${dist_dir}/"
cp "${project_root}"/*.png "${dist_dir}/"
cp "${project_root}"/*.wav "${dist_dir}/"
cp "${project_root}"/*.ogg "${dist_dir}/"

echo "Web build completed: ${dist_dir}"
