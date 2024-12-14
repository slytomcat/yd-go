#!/bin/bash
### build script

CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=v.$(git branch --show-current)-$(git rev-parse --short HEAD)"

if which upx >/dev/null 2>&1
then
   upx -qqq --best yd-go
else
   echo "'upx' is not installed and binary is not compressed. If You need compressed binary then install 'upx' and rebuild."
fi
