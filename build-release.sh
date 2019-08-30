#!/bin/bash

export GO111MODULE=on
export GOOS=linux
export GOARCH=amd64

packr2 build -ldflags "-X main.Version=$(git rev-list -1 HEAD)"
packr2 clean