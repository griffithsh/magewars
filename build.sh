#!/usr/bin/env bash
set -euo pipefail

rm -rf ./dist/
mkdir -p ./dist/

go build -o ./dist/magewars-server ./cmd/magewars-server
GOOS=js GOARCH=wasm go build -o ./dist/magewars.wasm ./cmd/magewars
