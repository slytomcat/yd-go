#!/bin/bash 
### build script

##### temporary path, see https://gitlab.xfce.org/xfce/xfce4-panel/-/issues/582 and https://github.com/godbus/dbus/issues/327
PATH_TO_PATCH="$(go env GOMODCACHE)/$(cat go.mod | grep 'github.com/godbus/dbus/v5' | sed 's/\s*\(\S\+\) \(.*\)$/\1@\2/')/conn.go"
patch <conn.patch $PATH_TO_PATCH


CGO_ENABLED=0 go build -ldflags "-X main.version=$(git branch --show-current)-$(git rev-parse --short HEAD)"
