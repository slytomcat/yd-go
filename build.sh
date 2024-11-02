#!/bin/bash 

die() {
    local message=$1
    echo "$message" >&2
    exit 1
}

GO111MODULE=on CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=v.$(git branch --show-current)-$(git rev-parse --short HEAD)" || die 'Sorry, build just failed.'

if type "upx" > /dev/null; then
  upx -qqq --best yd-go
else
  echo "Executable is uncompressed due to 'upx' command was not found. You need to install it and re-run the script again."
fi

