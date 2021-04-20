#!/bin/bash 
### build script

go build -ldflags "-X main.version=$(git describe --tag)" 