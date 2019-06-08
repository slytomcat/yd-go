#!/usr/bin/env bash
#

# check that this is the release
export TAG=$(git describe --abbrev=0 --tags); 
#if [[ $TAG != $(git describe --tags) ]]; then 
#  # exit if it is not release
#  exit 0
#fi

# get upload assets utility
wget https://gist.githubusercontent.com/stefanbuck/ce788fee19ab6eb0b4447a85fc99f447/raw/dbadd7d310ce8446de89c4ffdf1db0b400d0f6c3/upload-github-release-asset.sh
chmod a+x upload-github-release-asset.sh

# install requirements
sudo apt install libgtk-3-dev libappindicator3-dev

# set environment
export OWNER=$CIRCLE_PROJECT_USERNAME
export REPO=$CIRCLE_PROJECT_REPONAME

# build binary for linux amd64 platform 
export GOOS=linux
export GOARCH=amd64
go build
mv yd-go yd-go-amd64
./upload-github-release-asset.sh github_api_token=$GHAPITOKEN owner=$OWNER repo=$REPO tag="$TAG" filename=yd-go-amd64

# build binary for linux 386 platform
export GOARCH=386
go build 
mv yd-go yd-go-386
./upload-github-release-asset.sh  github_api_token=$GHAPITOKEN owner=$OWNER repo=$REPO tag=$TAG filename=yd-go-386
