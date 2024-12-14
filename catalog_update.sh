#!/bin/bash

go install golang.org/x/text/cmd/gotext@latest
gotext update -out catalog.go
