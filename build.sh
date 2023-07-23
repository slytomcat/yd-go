#!/bin/bash 
### build script

CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=v.$(git branch --show-current)-$(git rev-parse --short HEAD)"
upx -qqq --best yd-go

