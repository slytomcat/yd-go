#!/bin/bash 
### build script

CGO_ENABLED=0 go build -ldflags "-X main.version=$(git describe --tag)" 