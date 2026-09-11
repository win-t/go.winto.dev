#!/bin/sh

for os in linux darwin; do
  for arch in amd64 arm64; do
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o tssh-$os-$arch -ldflags='-s -w' -tags ts_debug_websockets,ts_omit_logtail .
  done
done
